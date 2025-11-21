// gateway_test.go - ゲートウェイの統合テスト（ION環境が必要）
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestManualDTNFlow(t *testing.T) {
	t.Skip("このテストは実際のION環境が必要なため、スキップします")

	// ION CLIゲートウェイを使用したテスト例
	gw := NewIonCLIGateway("example.com", 80, 30*time.Second)
	defer gw.GetUnsolicitedResponseCh() // チャンネルを消費

	// テストリクエスト作成
	reqBody := "This is a test request from Go test"
	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"http://example.com/api/test",
		strings.NewReader(reqBody),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	fmt.Println(">>> Starting DTN Send/Receive Test >>>")

	// BpRequestを作成してテスト（実装は省略）
	// breq := &model.BpRequest{...}
	// resp, err := gw.ProxyRequest(context.Background(), breq)

	fmt.Println("Test skipped - requires ION environment")
}
