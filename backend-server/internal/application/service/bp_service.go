package service

import (
	"bytes"
	"io"
	"log"
	"net/http"

	gateway_interfaces "github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/gateway/gateway_interfaces"
	repository_interfaces "github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/repository/repository_interfaces"
	scheduler_interfaces "github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/interface/scheduler"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/utils"
)

type BpService struct {
	bpgateway    gateway_interfaces.BpGateway
	bprepository repository_interfaces.BpRepository
	bpscheduler  scheduler_interfaces.BpScheduler
}

func NewBpService(
	bpgateway gateway_interfaces.BpGateway,
	bprepository repository_interfaces.BpRepository,
	bpscheduler scheduler_interfaces.BpScheduler,
) *BpService {
	return &BpService{
		bpgateway:    bpgateway,
		bprepository: bprepository,
		bpscheduler:  bpscheduler,
	}
}

// ProxyRequest HTTPリクエストを転送する（キャッシュ可能な場合はキャッシュもチェック）
func (bs *BpService) ProxyRequest(breq *model.BpRequest) (*model.BpResponse, error) {
	// キャッシュ不可の場合は直接転送
	if !breq.IsCacheable() {
		return bs.bpgateway.ProxyRequest(breq)
	}

	// キャッシュ可能な場合はキャッシュから取得を試みる
	cacheKey := breq.GenerateCacheKey()
	cachedResp, found, err := bs.bprepository.GetResponse(cacheKey)
	if err != nil {
		// キャッシュ取得エラー: Gateway層で直接転送
		return bs.bpgateway.ProxyRequest(breq)
	}

	if found {
		// キャッシュヒット: キャッシュされたレスポンスを返す
		return cachedResp, nil
	}

	// キャッシュミス: cronジョブにリクエストを予約してデフォルトページを返す
	return bs.bpscheduler.DownloadPage(breq)
}

// convertHTTPResponseToBpResponse HTTPレスポンスをBpResponseに変換
func convertHTTPResponseToBpResponse(resp *http.Response) *model.BpResponse {
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	// レスポンスボディを再度読み取れるようにするため、新しいReaderを作成
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return &model.BpResponse{
		StatusCode:    resp.StatusCode,
		Headers:       resp.Header,
		Body:          bodyBytes,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
	}
}

// convertBpResponseToHTTPResponse BpResponseをHTTPレスポンスに変換
func convertBpResponseToHTTPResponse(bpResp *model.BpResponse) *http.Response {
	resp := &http.Response{
		StatusCode: bpResp.StatusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(bpResp.Body)),
	}

	// ヘッダーをコピー
	for key, values := range bpResp.Headers {
		for _, value := range values {
			resp.Header.Add(key, value)
		}
	}

	resp.ContentLength = bpResp.ContentLength
	return resp
}

// createDefaultPageResponse デフォルトページ（処理中ページ）のBpResponseを作成
// pages/default.txtからHTMLを読み込む
func createDefaultPageResponse() *model.BpResponse {
	// pages/default.txtからHTMLを読み込む
	htmlBytes, err := utils.LoadDefaultPage()
	if err != nil {
		log.Printf("Failed to load default page: %v", err)
		// エラーが発生した場合は空のレスポンスを返す
		htmlBytes = []byte("")
	}

	headers := make(map[string][]string)
	headers["Content-Type"] = []string{"text/html; charset=utf-8"}
	headers["Cache-Control"] = []string{"no-cache, no-store, must-revalidate"}
	headers["Pragma"] = []string{"no-cache"}
	headers["Expires"] = []string{"0"}

	return &model.BpResponse{
		StatusCode:    http.StatusOK,
		Headers:       headers,
		Body:          htmlBytes,
		ContentType:   "text/html; charset=utf-8",
		ContentLength: int64(len(htmlBytes)),
	}
}
