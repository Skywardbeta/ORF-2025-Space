package model

import "time"

// CacheMetadata キャッシュのメタデータ（Redisに保存）
type CacheMetadata struct {
	// FilePath ファイルシステム上のファイルパス
	FilePath string `json:"file_path"`

	// StatusCode HTTPステータスコード
	StatusCode int `json:"status_code"`

	// Headers HTTPヘッダー
	Headers map[string][]string `json:"headers"`

	// ContentType Content-Typeヘッダーの値
	ContentType string `json:"content_type,omitempty"`

	// ContentLength Content-Lengthヘッダーの値
	ContentLength int64 `json:"content_length,omitempty"`

	// CreatedAt キャッシュ作成時刻
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt キャッシュ有効期限
	ExpiresAt time.Time `json:"expires_at"`
}

// IsExpired キャッシュが有効期限切れかどうかを判定する（domain層のロジック）
// 現在時刻の取得もdomain層で隠蔽される
func (cm *CacheMetadata) IsExpired() bool {
	return time.Now().After(cm.ExpiresAt)
}
