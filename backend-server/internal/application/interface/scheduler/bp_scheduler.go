package scheduler_interfaces

import (
	"github.com/watanabetatsumi/ORF-2025-Space/backend-server/internal/application/model"
)

type BpScheduler interface {
	DownloadPage(req *model.BpRequest) (*model.BpResponse, error)
}
