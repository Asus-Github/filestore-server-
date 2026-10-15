package process

import (
	"encoding/json"
	"log"

	"filestore-server/mq"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/store/oss"
)

// Transfer : 处理文件转移, 返回是否处理成功
func Transfer(msg []byte) bool {
	log.Println(string(msg))

	pubData := mq.TransferData{}
	if err := json.Unmarshal(msg, &pubData); err != nil {
		log.Println(err.Error())
		return false
	}

	bucket, err := oss.Bucket()
	if err != nil {
		log.Println(err.Error())
		return false
	}
	// PutObjectFromFile 内部负责打开/关闭本地文件
	if err := bucket.PutObjectFromFile(pubData.DestLocation, pubData.CurLocation); err != nil {
		log.Println(err.Error())
		return false
	}

	resp, err := dbcli.UpdateFileLocation(pubData.FileHash, pubData.DestLocation)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	if !resp.Suc {
		log.Println("更新数据库异常，请检查:" + pubData.FileHash)
		return false
	}
	return true
}
