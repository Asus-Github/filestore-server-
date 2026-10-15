package rpc

import (
	"context"

	cmn "filestore-server/common"
	cfg "filestore-server/config"
	dlProto "filestore-server/service/download/proto"
)

// Download :download结构体
type Download struct{}

// DownloadEntry : 获取下载入口(未配置LB地址时返回空, 由网关按访问域名拼接)
func (u *Download) DownloadEntry(
	ctx context.Context,
	req *dlProto.ReqEntry,
	res *dlProto.RespEntry) error {

	res.Code = cmn.StatusOK
	res.Entry = cfg.DownloadLBHost
	return nil
}
