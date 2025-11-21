// bp_socket_gateway_test.go - BP-Socketゲートウェイのユニットテスト
package gateway

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

func TestDTNJsonSerialization(t *testing.T) {
	req := &model.BpRequest{
		Method:  "GET",
		URL:     "https://example.com/test",
		Headers: map[string][]string{"User-Agent": {"test"}},
		Body:    []byte("test body"),
	}

	dtnReq := NewDTNJsonRequest("test-id", req)

	if dtnReq.Version != protocolVersion {
		t.Errorf("Expected version %d, got %d", protocolVersion, dtnReq.Version)
	}

	if dtnReq.RequestID != "test-id" {
		t.Errorf("Expected request ID 'test-id', got '%s'", dtnReq.RequestID)
	}

	if dtnReq.Method != "GET" {
		t.Errorf("Expected method GET, got %s", dtnReq.Method)
	}

	jsonData, err := json.Marshal(dtnReq)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DTNJsonRequest
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.RequestID != dtnReq.RequestID {
		t.Errorf("RequestID mismatch after round-trip")
	}
}

func TestDTNJsonResponseConversion(t *testing.T) {
	dtnResp := &DTNJsonResponse{
		Version:       protocolVersion,
		RequestID:     "test-id",
		StatusCode:    200,
		Headers:       map[string][]string{"Content-Type": {"application/json"}},
		Body:          "dGVzdA==", // "test" in base64
		ContentType:   "application/json",
		ContentLength: 4,
	}

	bpResp, err := ConvertToBpResponse(dtnResp)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if bpResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", bpResp.StatusCode)
	}

	if string(bpResp.Body) != "test" {
		t.Errorf("Expected body 'test', got '%s'", string(bpResp.Body))
	}

	if bpResp.ContentType != "application/json" {
		t.Errorf("Expected content-type 'application/json', got '%s'", bpResp.ContentType)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if len(id1) == 0 {
		t.Error("Generated ID should not be empty")
	}
}

func TestResponseChannelMapping(t *testing.T) {
	t.Skip("Requires actual bp-socket environment")
}

func TestBpSocketGatewayLinuxOnly(t *testing.T) {
	_, err := NewBpSocketGateway(149, 1, 150, 1, 30*time.Second)

	// Linux以外のプラットフォームでは失敗する（bp-socketはLinux専用）
	if err != nil {
		t.Logf("Expected error on non-Linux: %v", err)
	}
}

func TestProtocolVersionCheck(t *testing.T) {
	req := &model.BpRequest{
		Method: "POST",
		URL:    "https://example.com",
		Body:   []byte("data"),
	}

	dtnReq := NewDTNJsonRequest("id-123", req)

	jsonData, _ := json.Marshal(dtnReq)

	var decoded DTNJsonRequest
	json.Unmarshal(jsonData, &decoded)

	if decoded.Version != protocolVersion {
		t.Errorf("Version mismatch: expected %d, got %d", protocolVersion, decoded.Version)
	}
}

func TestUnsolicitedResponse(t *testing.T) {
	t.Skip("Requires actual bp-socket environment and daemon")
}

func TestConnectionReconnect(t *testing.T) {
	t.Skip("Requires actual bp-socket environment")

	// 再接続ロジックの検証
}

func TestMaxBundleSize(t *testing.T) {
	if maxBundleSize != 4*1024*1024 {
		t.Errorf("Expected maxBundleSize to be 4MB, got %d", maxBundleSize)
	}
}

func TestContextCancellation(t *testing.T) {
	t.Skip("Requires actual bp-socket environment")
}
