package gateway

import (
	"log"

	"github.com/robfig/cron/v3"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/utils"
)

type BpGateway struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	cron *cron.Cron
}

func NewBpGateway(host string, port int, cron *cron.Cron) *BpGateway {
	return &BpGateway{
		Host: host,
		Port: port,
		cron: cron,
	}
}

func (g *BpGateway) ProxyRequest(req *model.BpRequest) (*model.BpResponse, error) {
	g.cron.Start()
	log.Println("バッチ処理スケジューラーを開始しました")

	// pages/default.txtからHTMLを読み込む
	htmlBytes, err := utils.LoadDefaultPage()
	if err != nil {
		return nil, err
	}

	return &model.BpResponse{
		StatusCode:    200,
		Headers:       req.Headers,
		Body:          htmlBytes,
		ContentType:   "text/html; charset=utf-8",
		ContentLength: int64(len(htmlBytes)),
	}, nil
}
