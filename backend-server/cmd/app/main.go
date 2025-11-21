package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/cmd/config"
	gateway_interface "github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/gateway"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/service"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/handlers"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/infrastructure/gateway"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/infrastructure/repository"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/infrastructure/repository/plugins"
	scheduler_worker "github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/infrastructure/worker"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/middleware"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/middleware/module"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/scheduler"
)

func main() {
	// ============================================
	// 設定の読み込み
	// ============================================
	conf := config.LoadConfig()

	// ============================================
	// 設定とインフラストラクチャの初期化
	// ============================================

	// Redisクライアントの初期化
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", conf.RedisClient.Host, conf.RedisClient.Port),
		Password: conf.RedisClient.Password,
		DB:       conf.RedisClient.DB,
	})
	redisConfig := plugins.RedisClientConfig{
		ReservedRequestsKey: conf.RedisKeys.ReservedRequestsKey,
		CacheMetaPattern:    conf.RedisKeys.CacheMetaPattern,
		ScanCount:           conf.RedisKeys.ScanCount,
	}
	repoClient := plugins.NewRedisClient(redisClient, redisConfig)

	// 依存関係の初期化: トランスポートモードに応じてゲートウェイを選択
	var bpgw gateway_interface.BpGateway
	switch conf.BPGateway.TransportMode {
	case "bp_socket":
		log.Printf("Using bp-socket transport (ipn:%d.%d -> ipn:%d.%d)",
			conf.BPGateway.BpSocket.LocalNodeNum,
			conf.BPGateway.BpSocket.LocalServiceNum,
			conf.BPGateway.BpSocket.RemoteNodeNum,
			conf.BPGateway.BpSocket.RemoteServiceNum)
		var err error
		bpgw, err = gateway.NewBpSocketGateway(
			conf.BPGateway.BpSocket.LocalNodeNum,
			conf.BPGateway.BpSocket.LocalServiceNum,
			conf.BPGateway.BpSocket.RemoteNodeNum,
			conf.BPGateway.BpSocket.RemoteServiceNum,
			conf.BPGateway.Timeout,
		)
		if err != nil {
			log.Fatalf("Failed to initialize BpSocketGateway: %v", err)
		}
	case "ion_cli":
		log.Printf("Using ION CLI transport (host=%s, port=%d)", conf.BPGateway.Host, conf.BPGateway.Port)
		bpgw = gateway.NewIonCLIGateway(conf.BPGateway.Host, conf.BPGateway.Port, conf.BPGateway.Timeout)
	default:
		log.Fatalf("Invalid transport mode: %s (use 'ion_cli' or 'bp_socket')", conf.BPGateway.TransportMode)
	}

	// デバッグモードの場合はローカルHTTPゲートウェイを使用
	if conf.Server.Mode == config.DebugMode {
		log.Println("Debug mode enabled: Using Local HTTP Gateway")
		bpgw = gateway.NewLocalGateway(conf.BPGateway.Timeout)
	}

	bprepo := repository.NewBpRepository(repoClient, conf.Cache.Dir)

	// ============================================
	// ミドルウェアの初期化
	// ============================================

	ssl_bump_app, err := module.NewSSLBumpHandler(conf.Middlware.CertPath, conf.Middlware.KeyPath, conf.Middlware.MaxCacheSize)
	if err != nil {
		log.Fatalf("Failed to initialize SSLBumpHandler: %v", err)
		return
	}
	middlwares := middleware.NewMiddlewarePlugins(
		ssl_bump_app,
	)

	// ============================================
	// アプリケーション層の初期化
	// ============================================

	bpsrv := service.NewBpService(bpgw, bprepo, conf.Server.DefaultDir, conf.Server.DefaultFileName)
	bpHandler := handlers.NewBpHandler(bpsrv, middlwares)

	// ============================================
	// サーバーのセットアップ
	// ============================================

	// gin.Default() の代わりに gin.New() を使用してカスタムロガーを設定
	r := gin.New()
	r.Use(gin.Recovery())

	// セキュリティヘッダーを追加するミドルウェア
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Next()
	})

	// カスタム Logger: CONNECT メソッドの場合はログを出力しない
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if param.Request.Method == http.MethodConnect {
			// CONNECT のログは捨てる
			return ""
		}

		// それ以外はデフォルトとほぼ同じ形式で出す
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s  %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
		)
	}))

	// 管理用エンドポイント: キャッシュの一括削除
	r.POST("/system/admin/cache/cleanup", func(c *gin.Context) {
		ctx := c.Request.Context()
		err := bprepo.DeleteAllCaches(ctx)
		if err != nil {
			c.JSON(500, gin.H{
				"error":   "Failed to cleanup expired cache",
				"message": err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"message": "Expired cache cleanup completed successfully",
		})
	})

	// CONNECTメソッドを処理するミドルウェアを追加
	// CONNECTメソッドのリクエストは、パスがhost:port形式になる可能性があるため、
	// NoRouteの前に処理する必要がある
	r.Use(func(c *gin.Context) {
		if c.Request.Method == "CONNECT" {
			bpHandler.GetContent(c)
			c.Abort()
			return
		}
		c.Next()
	})

	// すべてのHTTPメソッド（GET、POST、PUT、DELETE、PATCHなど）とパスに対応
	// NoRouteは既存のルートにマッチしないすべてのリクエストを処理する
	r.NoRoute(bpHandler.GetContent)

	// ============================================
	// Worker Poolの起動（非同期リクエスト処理）
	// ============================================
	// プラグイン可能なWorker実装を使用
	reqHandler := scheduler_worker.NewRequestHandler(bprepo, bpgw, conf.Cache.DefaultTTL)
	queueWatcher := scheduler_worker.NewQueueWatcher(bprepo, conf.Worker.QueueWatchTimeout)
	cacheHandler := scheduler_worker.NewCacheHandler(bprepo)
	processor := scheduler.NewRequestProcessor(conf.Worker.Workers, reqHandler, queueWatcher, cacheHandler, conf.Cache.CleanupInterval) // 5つのworker
	ctx := context.Background()
	processor.Start(ctx)

	// ============================================
	// HTTPサーバーの起動
	// ============================================
	addr := fmt.Sprintf(":%d", conf.Server.Port)
	log.Printf("HTTPサーバーを起動します... (ポート: %d)", conf.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
