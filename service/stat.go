package services

import (
	"sync/atomic"

	nft_proxy "github.com/alphabatem/nft-proxy"
	"github.com/babilu-online/common/context"
)

type StatService struct {
	context.DefaultService

	imageFilesServed uint64
	mediaFilesServed uint64
	requestsServed   uint64

	sql *SqliteService
}

const STAT_SVC = "stat_svc"

func (svc StatService) Id() string {
	return STAT_SVC
}

func (svc *StatService) Start() error {
	svc.sql = svc.Service(SQLITE_SVC).(*SqliteService)

	return nil
}

func (svc *StatService) IncrementImageFileRequests() {
	atomic.AddUint64(&svc.imageFilesServed, 1)
}

func (svc *StatService) IncrementMediaFileRequests() {
	atomic.AddUint64(&svc.mediaFilesServed, 1)
}

func (svc *StatService) IncrementMediaRequests() {
	atomic.AddUint64(&svc.requestsServed, 1)
}

func (svc *StatService) ServiceStats() (map[string]interface{}, error) {
	var imgCount int64
	svc.sql.Db().Model(&nft_proxy.SolanaMedia{}).Count(&imgCount)

	return map[string]interface{}{
		"images_stored":      imgCount,
		"requests_served":    svc.requestsServed,
		"image_files_served": svc.imageFilesServed,
		"media_files_served": svc.mediaFilesServed,
	}, nil
}
