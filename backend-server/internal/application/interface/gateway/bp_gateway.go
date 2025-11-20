package gateway

import (
	"context"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

// BpGateway HTTPリクエストを転送するためのゲートウェイインターフェース
type BpGateway interface {
	// ProxyRequest HTTPリクエストを転送先に送信する
	// ctx: コンテキスト（リクエストのキャンセレーションやタイムアウト制御に使用）
	ProxyRequest(ctx context.Context, req *model.BpRequest) (*model.BpResponse, error)
}
