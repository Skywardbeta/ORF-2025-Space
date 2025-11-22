// ion_cli_gateway.go - IONのbpsendfile/bprecvfileコマンドを使用するゲートウェイ実装
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type IonCLIGateway struct {
	Host                  string
	Port                  int
	Timeout               time.Duration
	responseChs           sync.Map
	UnsolicitedResponseCh chan *model.BpResponse
}

func NewIonCLIGateway(host string, port int, timeout time.Duration) *IonCLIGateway {
	g := &IonCLIGateway{
		Host:                  host,
		Port:                  port,
		Timeout:               timeout,
		UnsolicitedResponseCh: make(chan *model.BpResponse, 100),
	}
	g.startReceiver()
	return g
}

func (g *IonCLIGateway) GetUnsolicitedResponseCh() <-chan *model.BpResponse {
	return g.UnsolicitedResponseCh
}

func (g *IonCLIGateway) startReceiver() {
	go func() {
		recvEID := "ipn:149.2"
		targetFile := "testfile1"

		for {
			if _, err := os.Stat(targetFile); err == nil {
				_ = os.Remove(targetFile)
			}

			log.Printf("[IonCLI] Waiting for response at %s...", recvEID)

			cmdRecv := exec.Command("bprecvfile", recvEID, "1")
			if err := cmdRecv.Run(); err != nil {
				log.Printf("[IonCLI] bprecvfile error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if _, err := os.Stat(targetFile); os.IsNotExist(err) {
				log.Printf("[IonCLI] file %s not found", targetFile)
				continue
			}

			fileContent, err := ioutil.ReadFile(targetFile)
			if err != nil {
				log.Printf("[IonCLI] read error: %v", err)
				_ = os.Remove(targetFile)
				continue
			}

			log.Printf("[IonCLI] Received: %s", string(fileContent))
			_ = os.Remove(targetFile)

			var dtnResp DTNJsonResponse
			if err := json.Unmarshal(fileContent, &dtnResp); err != nil {
				log.Printf("[IonCLI] JSON parse error: %v", err)
				continue
			}

			g.dispatchResponse(&dtnResp)
		}
	}()
}

func (g *IonCLIGateway) dispatchResponse(dtnResp *DTNJsonResponse) {
	if ch, ok := g.responseChs.Load(dtnResp.RequestID); ok {
		log.Printf("[IonCLI] Dispatching response for ID: %s", dtnResp.RequestID)
		select {
		case ch.(chan *DTNJsonResponse) <- dtnResp:
		default:
			log.Printf("[IonCLI] Channel blocked for ID: %s", dtnResp.RequestID)
		}
	} else {
		log.Printf("[IonCLI] Unsolicited response ID: %s", dtnResp.RequestID)
		bpResp, err := ConvertToBpResponse(dtnResp)
		if err != nil {
			log.Printf("[IonCLI] Convert error: %v", err)
			return
		}

		select {
		case g.UnsolicitedResponseCh <- bpResp:
			log.Printf("[IonCLI] Dispatched unsolicited response")
		default:
			log.Printf("[IonCLI] Unsolicited channel full")
		}
	}
}

func (g *IonCLIGateway) ProxyRequest(ctx context.Context, breq *model.BpRequest) (*model.BpResponse, error) {
	reqID := generateID()

	respCh := make(chan *DTNJsonResponse, 1)
	g.responseChs.Store(reqID, respCh)
	defer func() {
		g.responseChs.Delete(reqID)
		close(respCh) // sendBundleでエラーが発生した場合でもチャネルを閉じる
	}()

	if err := g.sendBundle(reqID, breq); err != nil {
		return nil, fmt.Errorf("bundle送信失敗: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, g.Timeout)
	defer cancel()

	select {
	case dtnResp := <-respCh:
		return ConvertToBpResponse(dtnResp)
	case <-ctx.Done():
		return nil, fmt.Errorf("request timeout or cancelled: %w", ctx.Err())
	}
}

func (g *IonCLIGateway) sendBundle(reqID string, breq *model.BpRequest) error {
	requestDir := "./request"

	if _, err := os.Stat(requestDir); os.IsNotExist(err) {
		if err := os.Mkdir(requestDir, 0755); err != nil {
			return fmt.Errorf("dir creation error: %w", err)
		}
	}

	filename := fmt.Sprintf("req_%s.txt", reqID)
	filePath := filepath.Join(requestDir, filename)

	dtnReq := NewDTNJsonRequest(reqID, breq)

	jsonData, err := json.Marshal(dtnReq)
	if err != nil {
		return fmt.Errorf("JSON marshal error: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("file write error: %w", err)
	}
	log.Printf("[IonCLI] Created file: %s (ID: %s)", filePath, reqID)

	cmdSend := exec.Command("bpsendfile", "ipn:149.1", "ipn:150.1", filePath)
	output, err := cmdSend.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bpsendfile error: %v, output: %s", err, string(output))
	}
	log.Printf("[IonCLI] bpsendfile output: %s", string(output))

	return nil
}
