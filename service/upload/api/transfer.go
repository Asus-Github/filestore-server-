package api

import (
	"encoding/json"

	cmn "filestore-server/common"
	cfg "filestore-server/config"
	"filestore-server/mq"
)

// publishTransfer : 写入异步转移任务, 由transfer服务将文件转移到OSS
func publishTransfer(filehash, curLocation, destLocation string) bool {
	data := mq.TransferData{
		FileHash:      filehash,
		CurLocation:   curLocation,
		DestLocation:  destLocation,
		DestStoreType: cmn.StoreOSS,
	}
	pubData, err := json.Marshal(data)
	if err != nil {
		return false
	}
	return mq.Publish(cfg.TransExchangeName, cfg.TransOSSRoutingKey, pubData)
}
