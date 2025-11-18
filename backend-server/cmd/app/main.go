package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/handlers"
	"github.com/watanabetatsumi/backend-server/cmd/config"
	"github.com/watanabetatsumi/backend-server/infrastructure/gateway"
	"github.com/watanabetatsumi/backend-server/intenal/service"
)

func main() {
	r := gin.Default()

	// // 1. 設定の読み込み
	conf := config.LoadConfig()

	// // 2. データベースへの接続
	// db, err := ConnectDB()
	// if err != nil {
	// 	log.Fatalf("failed to connect to database: %v", err)
	// }
	// defer db.Close() // アプリケーション終了時にDB接続を閉じる

	// 3. Cronスケジューラーの作成
	// Cronスケジューラーの作成
	c := cron.New(cron.WithSeconds()) // 秒単位のスケジュールを有効化

	// バッチ処理の例1: 毎分実行
	_, err := c.AddFunc("0 * * * * *", func() {
		log.Printf("[Batch] 毎分実行されるバッチ処理: %s", time.Now().Format("2006-01-02 15:04:05"))
		// ここに実際のバッチ処理ロジックを記述
		// 例: データの同期、キャッシュのクリア、ログの整理など
	})
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}

	// プログラムを継続実行（Ctrl+Cで終了）
	select {}

	bpgw := gateway.NewBpGateway(
		conf.BPGatewayHost,
		conf.BPGatewayPort,
	)

	bpsrv := service.NewBpService(bpgw)

	bpHandler := handlers.NewBpHandler(bpsrv)

	r.GET("/bp_get/:id", bpHandler.GetContent)

	r.Run()
}
