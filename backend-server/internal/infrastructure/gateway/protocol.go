// protocol.go - BP通信用のJSON/Base64 serialization
package gateway

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

const protocolVersion = 1

type DTNJsonRequest struct {
	Version   int                 `json:"version"`
	RequestID string              `json:"request_id"`
	Method    string              `json:"method"`
	URL       string              `json:"url"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"`
}

type DTNJsonResponse struct {
	Version       int                 `json:"version"`
	RequestID     string              `json:"request_id"`
	StatusCode    int                 `json:"status_code"`
	Headers       map[string][]string `json:"headers"`
	Body          string              `json:"body"`
	ContentType   string              `json:"content_type"`
	ContentLength int64               `json:"content_length"`
}

func NewDTNJsonRequest(reqID string, breq *model.BpRequest) *DTNJsonRequest {
	return &DTNJsonRequest{
		Version:   protocolVersion,
		RequestID: reqID,
		Method:    breq.Method,
		URL:       breq.URL,
		Headers:   breq.Headers,
		Body:      base64.StdEncoding.EncodeToString(breq.Body),
	}
}

func ConvertToBpResponse(dtnResp *DTNJsonResponse) (*model.BpResponse, error) {
	decodedBodyBytes, err := base64.StdEncoding.DecodeString(dtnResp.Body)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}

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
