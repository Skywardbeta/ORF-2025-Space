// bp_socket_gateway.go - AF_BPソケットを直接使用するゲートウェイ実装（自動再接続対応）
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/infrastructure/gateway/bpsocket"
)

const maxBundleSize = 4 * 1024 * 1024

type BpSocketGateway struct {
	conn                  *bpsocket.Connection
	timeout               time.Duration
	responseChs           sync.Map
	UnsolicitedResponseCh chan *model.BpResponse
	stopCh                chan struct{}
	wg                    sync.WaitGroup
}

func NewBpSocketGateway(
	localNodeNum, localSvcNum,
	remoteNodeNum, remoteSvcNum uint64,
	timeout time.Duration,
) (*BpSocketGateway, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("bp-socket is only supported on Linux (current OS: %s)", runtime.GOOS)
	}

	conn, err := bpsocket.NewConnection(localNodeNum, localSvcNum, remoteNodeNum, remoteSvcNum)
	if err != nil {
		return nil, fmt.Errorf("BP connection failed: %w", err)
	}

	g := &BpSocketGateway{
		conn:                  conn,
		timeout:               timeout,
		UnsolicitedResponseCh: make(chan *model.BpResponse, 100),
		stopCh:                make(chan struct{}),
	}

	g.start()
	log.Printf("[BpSocket] Gateway started: %s -> ipn:%d.%d",
		conn.LocalAddr().String(), remoteNodeNum, remoteSvcNum)

	return g, nil
}

func (g *BpSocketGateway) GetUnsolicitedResponseCh() <-chan *model.BpResponse {
	return g.UnsolicitedResponseCh
}

func (g *BpSocketGateway) start() {
	g.wg.Add(1)
	go g.receiveLoop()
}

func (g *BpSocketGateway) Close() error {
	close(g.stopCh)
	// Recv()をブロック解除するため先にソケットをクローズ
	if err := g.conn.Close(); err != nil {
		log.Printf("[BpSocket] Error closing connection: %v", err)
	}
	g.wg.Wait()
	return nil
}

func (g *BpSocketGateway) receiveLoop() {
	defer g.wg.Done()

	buf := make([]byte, maxBundleSize)
	consecutiveErrors := 0

	for {
		select {
		case <-g.stopCh:
			log.Println("[BpSocket] Receive loop stopped")
			return
		default:
		}

		n, fromAddr, err := g.conn.Recv(buf)
		if err != nil {
			select {
			case <-g.stopCh:
				return
			default:
			}

			consecutiveErrors++
			log.Printf("[BpSocket] Recv error (%d): %v", consecutiveErrors, err)

			// 3回連続エラーで再接続を試行
			if consecutiveErrors >= 3 {
				log.Printf("[BpSocket] Too many errors, attempting reconnect")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := g.conn.Reconnect(ctx); err != nil {
					cancel()
					log.Printf("[BpSocket] Reconnect failed: %v, stopping receive loop", err)
					return
				}
				cancel()
				consecutiveErrors = 0
				log.Println("[BpSocket] Reconnect successful, resuming receive")
			} else {
				time.Sleep(time.Duration(consecutiveErrors) * time.Second)
			}
			continue
		}

		consecutiveErrors = 0

		// 切断検出（バッファ上限に達した場合）
		if n >= maxBundleSize {
			log.Printf("[BpSocket] WARNING: Received %d bytes (buffer limit), possible truncation", n)
		}

		log.Printf("[BpSocket] Received %d bytes from %s", n, fromAddr.String())

		var dtnResp DTNJsonResponse
		if err := json.Unmarshal(buf[:n], &dtnResp); err != nil {
			log.Printf("[BpSocket] JSON unmarshal error: %v", err)
			continue
		}

		if dtnResp.Version != protocolVersion {
			log.Printf("[BpSocket] Protocol version mismatch: got %d, expected %d",
				dtnResp.Version, protocolVersion)
		}

		g.dispatchResponse(&dtnResp)
	}
}

func (g *BpSocketGateway) dispatchResponse(dtnResp *DTNJsonResponse) {
	if ch, ok := g.responseChs.Load(dtnResp.RequestID); ok {
		log.Printf("[BpSocket] Dispatching response for ID: %s", dtnResp.RequestID)
		select {
		case ch.(chan *DTNJsonResponse) <- dtnResp:
		default:
			log.Printf("[BpSocket] Channel blocked for ID: %s", dtnResp.RequestID)
		}
	} else {
		log.Printf("[BpSocket] Unsolicited response ID: %s", dtnResp.RequestID)
		bpResp, err := ConvertToBpResponse(dtnResp)
		if err != nil {
			log.Printf("[BpSocket] Convert error: %v", err)
			return
		}

		select {
		case g.UnsolicitedResponseCh <- bpResp:
			log.Printf("[BpSocket] Dispatched unsolicited response")
		default:
			log.Printf("[BpSocket] Unsolicited channel full")
		}
	}
}

func (g *BpSocketGateway) ProxyRequest(ctx context.Context, breq *model.BpRequest) (*model.BpResponse, error) {
	reqID := generateID()

	respCh := make(chan *DTNJsonResponse, 1)
	g.responseChs.Store(reqID, respCh)
	defer g.responseChs.Delete(reqID)

	if err := g.sendBundle(ctx, reqID, breq); err != nil {
		return nil, fmt.Errorf("bundle送信失敗: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	select {
	case dtnResp := <-respCh:
		return ConvertToBpResponse(dtnResp)
	case <-ctx.Done():
		return nil, fmt.Errorf("request timeout or cancelled: %w", ctx.Err())
	}
}

func (g *BpSocketGateway) sendBundle(ctx context.Context, reqID string, breq *model.BpRequest) error {
	dtnReq := NewDTNJsonRequest(reqID, breq)

	jsonData, err := json.Marshal(dtnReq)
	if err != nil {
		return fmt.Errorf("JSON marshal error: %w", err)
	}

	if len(jsonData) > maxBundleSize {
		return fmt.Errorf("bundle size %d exceeds max %d", len(jsonData), maxBundleSize)
	}

	log.Printf("[BpSocket] Sending bundle: ID=%s, size=%d bytes", reqID, len(jsonData))

	if err := g.conn.Send(ctx, jsonData); err != nil {
		return fmt.Errorf("socket send error: %w", err)
	}
	return nil
}
