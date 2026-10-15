package api

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	cmn "filestore-server/common"
	cfg "filestore-server/config"
	"filestore-server/middleware"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/store/ceph"
	"filestore-server/store/oss"
)

// localFilePath : 文件在本地存储中的路径
func localFilePath(filehash string) string {
	return cfg.TempLocalRootDir + filehash
}

// commitLocalFile : 将已校验的临时文件移动到本地存储目录(以hash命名), 返回最终路径
// 同一hash的文件已存在且大小一致时直接复用, 删除临时文件
func commitLocalFile(tmpPath, filehash string, size int64) (string, error) {
	dest := localFilePath(filehash)
	if fi, err := os.Stat(dest); err == nil && fi.Size() == size {
		os.Remove(tmpPath)
		return dest, nil
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		log.Println(err)
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return dest, nil
}

// storeFile : 按配置将本地文件同步或异步转存到Ceph/OSS, 返回最终的存储地址
func storeFile(localPath, filehash string) (string, error) {
	switch cfg.CurrentStoreType {
	case cmn.StoreCeph:
		// 文件写入Ceph存储
		data, err := os.ReadFile(localPath)
		if err != nil {
			return "", err
		}
		cephPath := cfg.CephRootDir + filehash
		if err := ceph.PutObject(cfg.CephBucket, cephPath, data); err != nil {
			return "", err
		}
		return cephPath, nil
	case cmn.StoreOSS:
		ossPath := cfg.OSSRootDir + filehash
		if cfg.AsyncTransferEnable {
			// 写入异步转移任务队列, 转移完成前文件仍可从本地下载
			if publishTransfer(filehash, localPath, ossPath) {
				return localPath, nil
			}
			log.Println("publish transfer task failed, fallback to sync upload:", filehash)
		}
		bucket, err := oss.Bucket()
		if err != nil {
			return "", err
		}
		if err := bucket.PutObjectFromFile(ossPath, localPath); err != nil {
			return "", err
		}
		return ossPath, nil
	default:
		return localPath, nil
	}
}

// finishUpload : 文件已落盘并校验通过, 转存并更新唯一文件表及用户文件表, 然后响应客户端
// 返回是否成功
func finishUpload(c *gin.Context, username string, fmeta dbcli.FileMeta) bool {
	location, err := storeFile(fmeta.Location, fmeta.FileSha1)
	if err != nil {
		log.Println("store file failed:", err)
		middleware.Fail(c, cmn.StatusUploadFailed, "文件存储失败, 请稍后重试")
		return false
	}
	fmeta.Location = location

	if _, err := dbcli.OnFileUploadFinished(fmeta); err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "保存文件信息失败, 请稍后重试")
		return false
	}
	if _, err := dbcli.OnUserFileUploadFinished(username, fmeta); err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "保存文件信息失败, 请稍后重试")
		return false
	}
	middleware.OK(c, "上传成功", gin.H{
		"FileHash": fmeta.FileSha1,
		"FileName": fmeta.FileName,
		"FileSize": fmeta.FileSize,
	})
	return true
}

// isLocalAddr : 文件是否存放在本地存储
func isLocalAddr(addr string) bool {
	return strings.HasPrefix(addr, cfg.TempLocalRootDir)
}

// fileAvailable : 唯一文件表中的文件是否真实可用(本地文件需存在且大小一致)
func fileAvailable(addr string, size int64) bool {
	if !isLocalAddr(addr) {
		return addr != ""
	}
	fi, err := os.Stat(addr)
	return err == nil && fi.Mode().IsRegular() && fi.Size() == size
}

// createTemp : 在指定目录下创建临时文件(目录不存在时自动创建)
func createTemp(dir, pattern string) (*os.File, error) {
	f, err := os.CreateTemp(dir, pattern)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(filepath.Clean(dir), 0755); err == nil {
			f, err = os.CreateTemp(dir, pattern)
		}
	}
	return f, err
}
