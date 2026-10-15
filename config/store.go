package config

import (
	"path/filepath"
	"time"

	cmn "filestore-server/common"
)

const (
	// CephRootDir : Ceph的存储路径prefix
	CephRootDir = "/ceph/"
	// OSSRootDir : OSS的存储路径prefix
	OSSRootDir = "oss/"
)

var (
	// TempLocalRootDir : 本地临时存储地址的路径(FS_DATA_DIR)
	TempLocalRootDir = absDir(envString("FS_DATA_DIR", "/data/fileserver/"))
	// TempPartRootDir : 分块文件在本地临时存储地址的路径(FS_PART_DIR)
	TempPartRootDir = absDir(envString("FS_PART_DIR", "/data/fileserver_part/"))
	// CurrentStoreType : 设置当前文件的存储类型(FS_STORE_TYPE: 1本地 2Ceph 3OSS)
	CurrentStoreType = cmn.StoreType(envInt64("FS_STORE_TYPE", int64(cmn.StoreLocal)))
	// MaxUploadSize : 普通上传接口允许的最大文件大小(字节), 更大的文件请使用分块上传
	MaxUploadSize = envInt64("FS_MAX_UPLOAD_SIZE", 1<<30)
	// ChunkSize : 分块上传的分块大小(字节)
	ChunkSize = envInt64("FS_CHUNK_SIZE", 5*1024*1024)
	// MultipartUploadTTL : 分块上传会话的有效期(期间可断点续传)
	MultipartUploadTTL = envDuration("FS_MPUPLOAD_TTL", 24*time.Hour)
)

// absDir : 转换为以"/"结尾的绝对路径, 保证落库的文件地址不依赖进程工作目录
func absDir(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return withSlash(dir)
}
