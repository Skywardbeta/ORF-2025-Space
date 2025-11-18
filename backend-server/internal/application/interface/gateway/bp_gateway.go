package gateway_interfaces

import (
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

// BpGateway HTTPリクエストを転送するためのゲートウェイインターフェース
type BpGateway interface {
	// ProxyRequest HTTPリクエストを転送先に送信する
	ProxyRequest(req *model.BpRequest) (*model.BpResponse, error)

	// ReserveRequest cronジョブで処理するためにリクエストを予約する
	// req: 予約するリクエスト
	ReserveRequest(req *model.BpRequest) (*model.BpResponse, error)
}
