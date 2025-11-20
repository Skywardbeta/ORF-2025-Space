package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type BpRepository struct {
	client   BpRepoClient
	cacheDir string
}

func NewBpRepository(client BpRepoClient, cacheDir string) *BpRepository {
	// キャッシュディレクトリが存在しない場合は作成
	_ = os.MkdirAll(cacheDir, 0755)

	return &BpRepository{
		client:   client,
		cacheDir: cacheDir,
	}
}

// GetResponse キャッシュからレスポンスを取得
func (br *BpRepository) GetResponse(ctx context.Context, cacheKey string) (*model.BpResponse, bool, error) {
	// Redisからメタデータを取得
	metaKey := _getMetaKey(cacheKey)
	metaData, err := br.client.GetMetaData(ctx, metaKey)
	if err != nil {
		return nil, false, nil // キャッシュミス
	}

	// メタデータが空の場合はキャッシュミス
	if len(metaData) == 0 {
		return nil, false, nil
	}

	// メタデータをデコード
	var metadata model.CacheMetadata
	err = json.Unmarshal(metaData, &metadata)
	if err != nil {
		// JSONデコードエラーの場合はキャッシュミスとして扱う（破損したキャッシュ）
		log.Printf("[BpRepository] GetResponse JSONデコードエラー: %v, metaKey=%s", err, metaKey)
		return nil, false, nil
	}

	// 有効期限チェック
	if metadata.IsExpired() {
		// TTLが切れている場合は削除
		_ = br.client.DeleteCache(ctx, cacheKey, metadata.FilePath)
		return nil, false, nil
	}

	// ファイルシステムからボディを読み込む
	body, err := os.ReadFile(metadata.FilePath)
	if err != nil {
		// ファイルが存在しない場合はRedisからも削除（アクセス時のクリア）
		if os.IsNotExist(err) {
			_ = br.client.DeleteCache(ctx, cacheKey, metadata.FilePath)
		}
		return nil, false, nil
	}

	// BpResponseを構築
	return &model.BpResponse{
		StatusCode:    metadata.StatusCode,
		Headers:       metadata.Headers,
		Body:          body,
		ContentType:   metadata.ContentType,
		ContentLength: metadata.ContentLength,
	}, true, nil
}

// SetResponseWithURL レスポンスをキャッシュに保存（URL指定版）
// BpRequestからキャッシュパス情報を生成してURLベースの階層構造でキャッシュを保存します
func (br *BpRepository) SetResponseWithURL(ctx context.Context, req *model.BpRequest, response *model.BpResponse, ttl time.Duration) error {
	// domain層のロジックを使用してキャッシュパス情報を生成
	pathInfo, err := req.GenerateCachePathInfo(response.ContentType)
	if err != nil {
		return fmt.Errorf("failed to generate cache path info: %w", err)
	}

	// ファイルパスを構築
	dirPath := filepath.Join(br.cacheDir, pathInfo.Host, pathInfo.Path)
	if pathInfo.SubDir != "" {
		dirPath = filepath.Join(dirPath, pathInfo.SubDir)
	}
	filePath := filepath.Join(dirPath, pathInfo.FileName)

	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(filePath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// ファイルシステムにボディを保存
	err = os.WriteFile(filePath, response.Body, 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// メタデータを作成
	now := time.Now()
	metadata := model.CacheMetadata{
		FilePath:      filePath,
		StatusCode:    response.StatusCode,
		Headers:       response.Headers,
		ContentType:   response.ContentType,
		ContentLength: response.ContentLength,
		CreatedAt:     now,
		ExpiresAt:     now.Add(ttl),
	}

	// メタデータをJSONにエンコード
	metaData, err := json.Marshal(metadata)
	if err != nil {
		// ファイルは保存済みなので削除
		_ = os.Remove(filePath)
		return err
	}

	// Redisにメタデータを保存（TTL付き）
	cacheKey := req.GenerateCacheKey()
	metaKey := _getMetaKey(cacheKey)
	err = br.client.SetMetaData(ctx, metaKey, metaData, ttl)
	if err != nil {
		// Redis保存に失敗した場合はファイルも削除
		_ = os.Remove(filePath)
		return err
	}

	return nil
}

// _getMetaKey メタデータ用のRedisキーを生成
func _getMetaKey(cacheKey string) string {
	return fmt.Sprintf("bp:cache:meta:%s", cacheKey)
}

// DeleteExpiredCaches 期限切れキャッシュを削除する
func (br *BpRepository) DeleteExpiredCaches(ctx context.Context) error {
	items, err := br.client.ScanExpiredCacheKeys(ctx)
	if err != nil {
		return err
	}

	// 各期限切れアイテムを削除
	for _, item := range items {
		_ = br.client.DeleteCache(ctx, item.Key, item.FilePath)
	}

	return nil
}

// ReserveRequest 非同期処理（Worker Pool）で処理するためにリクエストを予約する
// Redisキューに追加して、RequestProcessorが非同期で処理する
func (br *BpRepository) ReserveRequest(ctx context.Context, req *model.BpRequest) error {
	log.Printf("[BpRepository] ReserveRequest called: URL=%s", req.URL)

	// BpRequestをJSONにエンコード
	job, err := json.Marshal(req)
	if err != nil {
		log.Printf("[BpRepository] JSON Marshal エラー: %v", err)
		return err
	}

	log.Printf("[BpRepository] Redisキューに追加: URL=%s, job size=%d bytes", req.URL, len(job))

	// RedisのListに追加（キューとして使用）
	err = br.client.ReserveRequest(ctx, job)
	if err != nil {
		log.Printf("[BpRepository] ReserveRequest failed: %v", err)
		return err
	}

	log.Printf("[BpRepository] ReserveRequest succeeded: URL=%s", req.URL)
	return nil
}

// GetReservedRequests 予約されたリクエストのリストを取得する
func (br *BpRepository) GetReservedRequests(ctx context.Context) ([]*model.BpRequest, error) {
	// Redisから生のバイトデータのリストを取得
	dataList, err := br.client.GetReservedRequests(ctx)
	if err != nil {
		return nil, err
	}

	// JSONをデコード（repository層の責務）
	var requests []*model.BpRequest
	for _, data := range dataList {
		var req model.BpRequest
		err := json.Unmarshal(data, &req)
		if err != nil {
			// 不正なデータはスキップ
			continue
		}
		requests = append(requests, &req)
	}

	return requests, nil
}

// RemoveReservedRequest 予約されたリクエストを削除する
func (br *BpRepository) RemoveReservedRequest(ctx context.Context, req *model.BpRequest) error {
	// リクエストをJSONにエンコード
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	err = br.client.RemoveReservedRequest(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

// BLPopReservedRequest 予約されたリクエストをブロッキングで取得する
func (br *BpRepository) BLPopReservedRequest(ctx context.Context, timeout time.Duration) (*model.BpRequest, error) {
	// Redisから生のバイトデータを取得
	data, err := br.client.BLPopReservedRequest(ctx, timeout)
	if err != nil {
		return nil, err
	}
	if data == nil {
		// タイムアウト
		return nil, nil
	}

	// JSONをデコード（repository層の責務）
	var req model.BpRequest
	err = json.Unmarshal(data, &req)
	if err != nil {
		return nil, err
	}

	return &req, nil
}
