package gateway

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

// 送信JSONデータの構造定義
type DTNJsonRequest struct {
	RequestID string              `json:"request_id"`
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"` // Base64文字列
}

// 受信JSONデータの構造定義
type DTNJsonResponse struct {
	RequestID     string              `json:"request_id"`
	StatusCode    int                 `json:"status_code"`
	Headers       map[string][]string `json:"headers"`
	Body          string              `json:"body"` // Base64文字列
	ContentType   string              `json:"content_type"`
	ContentLength int64               `json:"content_length"`
}

type BpGateway struct {
	Host        string        `json:"host"`
	Port        int           `json:"port"`
	Timeout     time.Duration // 追加
	responseChs sync.Map      // map[string]chan *DTNJsonResponse

	// Push受信したレスポンスを通知するチャンネル（Service層が監視する）
	UnsolicitedResponseCh chan *model.BpResponse
}

func NewBpGateway(host string, port int, timeout time.Duration) *BpGateway {
	g := &BpGateway{
		Host:                  host,
		Port:                  port,
		Timeout:               timeout,                           // 追加
		UnsolicitedResponseCh: make(chan *model.BpResponse, 100), // バッファを持たせる
	}
	g.StartReceiver()
	return g
}

func (g *BpGateway) GetUnsolicitedResponseCh() <-chan *model.BpResponse {
	return g.UnsolicitedResponseCh
}

func (g *BpGateway) StartReceiver() {
	go func() {
		recvEID := "ipn:149.2"
		targetFile := "testfile1" // IONの仕様でファイル名が決まるため固定

		for {
			// 前回の受信ファイルが残っていたら削除
			if _, err := os.Stat(targetFile); err == nil {
				_ = os.Remove(targetFile)
			}

			log.Printf("[Receiver] Waiting for response bundle at %s...\n", recvEID)

			// bprecvfile を実行（受信完了までブロック）
			cmdRecv := exec.Command("bprecvfile", recvEID, "1")
			if err := cmdRecv.Run(); err != nil {
				log.Printf("[Receiver] error executing bprecvfile: %v", err)
				time.Sleep(1 * time.Second) // エラー時は少し待つ
				continue
			}

			// 受信ファイルの確認
			if _, err := os.Stat(targetFile); os.IsNotExist(err) {
				log.Printf("[Receiver] received file %s not found", targetFile)
				continue
			}

			// ファイル読み込み
			fileContent, err := ioutil.ReadFile(targetFile)
			if err != nil {
				log.Printf("[Receiver] error reading received file: %v", err)
				_ = os.Remove(targetFile)
				continue
			}

			log.Printf("[Receiver] Received content: %s", string(fileContent))

			// 処理後は即削除
			_ = os.Remove(targetFile)

			// レスポンス解析 (JSON)
			var dtnResp DTNJsonResponse
			if err := json.Unmarshal(fileContent, &dtnResp); err != nil {
				log.Printf("[Receiver] error parsing received JSON: %v", err)
				continue
			}

			// Dispatcher: IDに対応するチャンネルに送信
			if ch, ok := g.responseChs.Load(dtnResp.RequestID); ok {
				log.Printf("[Receiver] Dispatching response for ID: %s", dtnResp.RequestID)
				// 非ブロッキング送信 (念のため)
				select {
				case ch.(chan *DTNJsonResponse) <- &dtnResp:
				default:
					log.Printf("[Receiver] Channel blocked or closed for ID: %s", dtnResp.RequestID)
				}
			} else {
				log.Printf("[Receiver] No waiting channel found for ID: %s. Treating as unsolicited push.", dtnResp.RequestID)
				// 待ち受け先がない場合は、Push通知として処理する（Service層へ通知）
				// BpResponseに変換して送る
				bpResp, err := convertToBpResponse(&dtnResp)
				if err != nil {
					log.Printf("[Receiver] Failed to convert push response: %v", err)
					continue
				}

				select {
				case g.UnsolicitedResponseCh <- bpResp:
					log.Printf("[Receiver] Dispatched unsolicited response to service layer")
				default:
					log.Printf("[Receiver] Unsolicited response channel full, dropping response")
				}
			}
		}
	}()
}

func (g *BpGateway) ProxyRequest(ctx context.Context, breq *model.BpRequest) (*model.BpResponse, error) {
	// リクエストID生成
	reqID := generateID()

	// レスポンス待ち受けチャンネルの作成と登録
	respCh := make(chan *DTNJsonResponse, 1) // バッファ1で送信側のブロック回避
	g.responseChs.Store(reqID, respCh)
	defer g.responseChs.Delete(reqID)

	// バンドル送信
	// BpRequestを直接渡すことで、元のURLを維持する
	if err := sendBundle(reqID, breq); err != nil {
		return nil, fmt.Errorf("failed to send bundle: %w", err)
	}

	// レスポンス待機
	ctx, cancel := context.WithTimeout(ctx, g.Timeout)
	defer cancel()

	select {
	case dtnResp := <-respCh:
		return convertToBpResponse(dtnResp)

	case <-ctx.Done():
		return nil, fmt.Errorf("request timed out or cancelled: %w", ctx.Err())
	}
}

// sendBundle: BpRequestをJSON化してbpsendfileで送信する
func sendBundle(reqID string, breq *model.BpRequest) error {
	requestDir := "./request"

	// ディレクトリ作成
	if _, err := os.Stat(requestDir); os.IsNotExist(err) {
		if err := os.Mkdir(requestDir, 0755); err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}

	// ユニークなファイル名を作成
	filename := fmt.Sprintf("req_%s.txt", reqID)
	filePath := filepath.Join(requestDir, filename)

	// DTNJsonRequestを作成
	dtnReq := DTNJsonRequest{
		RequestID: reqID,
		Method:    breq.Method,
		URL:       breq.URL, // 元のURLをそのまま使用
		Headers:   breq.Headers,
		Body:      base64.StdEncoding.EncodeToString(breq.Body),
	}

	// JSONにマーシャル
	jsonData, err := json.Marshal(dtnReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request to JSON: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}
	log.Printf("[Sender] Created request file: %s (ID: %s)\n", filePath, reqID)

	// bpsendfile コマンド実行
	// 送信元: ipn:149.1 -> 宛先: ipn:150.1
	cmdSend := exec.Command("bpsendfile", "ipn:149.1", "ipn:150.1", filePath)
	output, err := cmdSend.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing bpsendfile: %v, output: %s", err, string(output))
	}
	log.Printf("[Sender] bpsendfile output: %s\n", string(output))

	return nil
}

// generateID creates a random hex string
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp if rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// convertToBpResponse converts DTNJsonResponse to model.BpResponse
func convertToBpResponse(dtnResp *DTNJsonResponse) (*model.BpResponse, error) {
	// Body (Base64) のデコード
	decodedBodyBytes, err := base64.StdEncoding.DecodeString(dtnResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 body: %v", err)
	}

	// ヘッダーの構築
	httpHeader := make(http.Header)
	for k, v := range dtnResp.Headers {
		for _, hVal := range v {
			httpHeader.Add(k, hVal)
		}
	}

	return &model.BpResponse{
		StatusCode:    dtnResp.StatusCode,
		Headers:       httpHeader,
		Body:          decodedBodyBytes,
		ContentType:   dtnResp.ContentType,
		ContentLength: dtnResp.ContentLength,
	}, nil
}
