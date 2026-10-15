package rpc

import (
	"context"

	cmn "filestore-server/common"
	cfg "filestore-server/config"
	upProto "filestore-server/service/upload/proto"
)

// Upload : upload结构体
type Upload struct{}

// UploadEntry : 获取上传入口(未配置LB地址时返回空, 由网关按访问域名拼接)
func (u *Upload) UploadEntry(
	ctx context.Context,
	req *upProto.ReqEntry,
	res *upProto.RespEntry) error {

	res.Code = cmn.StatusOK
	res.Entry = cfg.UploadLBHost
	return nil
}
