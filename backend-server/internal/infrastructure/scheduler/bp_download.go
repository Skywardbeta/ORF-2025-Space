package scheduler

import (
	"log"

	"github.com/robfig/cron/v3"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/utils"
)

type BpScheduler struct {
	cron *cron.Cron
}

func NewBpScheduler(cron *cron.Cron) *BpScheduler {
	return &BpScheduler{
		cron: cron,
	}
}

func (s *BpScheduler) DownloadPage(req *model.BpRequest) (*model.BpResponse, error) {
	s.cron.Start()
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
