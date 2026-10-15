package api

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	cmn "filestore-server/common"
	cfg "filestore-server/config"
	"filestore-server/middleware"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/util"
)

// multipartOverhead : multipart表单中除文件内容外其他字段的预留空间
const multipartOverhead = 1 << 20

// DoUploadHandler ： 处理文件上传(普通上传, 适合较小的文件)
// 表单字段: file(必填), filehash(选填, 提供时服务端会校验), filename(选填, 原始文件名)
func DoUploadHandler(c *gin.Context) {
	username := middleware.Username(c)

	// 1. 限制请求体大小, 从form表单中获得文件内容句柄
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxUploadSize+multipartOverhead)
	file, head, err := c.Request.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			middleware.Fail(c, cmn.StatusFileTooLarge, "文件过大, 请使用分块上传")
			return
		}
		log.Printf("Failed to get form data, err:%s\n", err.Error())
		middleware.Fail(c, cmn.StatusParamInvalid, "请选择要上传的文件")
		return
	}
	defer file.Close()

	// 浏览器会把multipart文件名中的 " 等字符转义为 %22, 因此优先使用表单中单独提交的原始文件名
	rawName := head.Filename
	if name := c.PostForm("filename"); name != "" {
		rawName = name
	}
	filename, err := util.SanitizeFileName(rawName)
	if err != nil {
		middleware.Fail(c, cmn.StatusParamInvalid, err.Error())
		return
	}

	// 2. 边写入临时文件边计算sha1, 避免将整个文件读入内存
	tmp, err := createTemp(cfg.TempLocalRootDir, ".upload-*")
	if err != nil {
		log.Printf("Failed to create file, err:%s\n", err.Error())
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}
	h := sha1.New()
	size, err := io.Copy(io.MultiWriter(tmp, h), file)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp.Name())
		log.Printf("Failed to save data into file, err:%s\n", err.Error())
		middleware.Fail(c, cmn.StatusUploadFailed, "上传失败")
		return
	}
	if size > cfg.MaxUploadSize {
		os.Remove(tmp.Name())
		middleware.Fail(c, cmn.StatusFileTooLarge, "文件过大, 请使用分块上传")
		return
	}

	// 3. 校验客户端提供的hash
	filehash := hex.EncodeToString(h.Sum(nil))
	if claimed := strings.ToLower(strings.TrimSpace(c.PostForm("filehash"))); claimed != "" && claimed != filehash {
		os.Remove(tmp.Name())
		middleware.Fail(c, cmn.StatusFileVerifyFailed, "文件校验失败, 内容与hash不一致")
		return
	}

	// 4. 将文件移动到存储位置
	location, err := commitLocalFile(tmp.Name(), filehash, size)
	if err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}

	// 5. 转存并更新文件表/用户文件表
	finishUpload(c, username, dbcli.FileMeta{
		FileSha1: filehash,
		FileName: filename,
		FileSize: size,
		Location: location,
	})
}

// TryFastUploadHandler : 尝试秒传接口
// 参数: filehash, filename, filesize(需与服务端文件大小一致)
func TryFastUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := middleware.Username(c)
	filehash := strings.ToLower(strings.TrimSpace(c.PostForm("filehash")))
	if !util.IsValidSha1(filehash) {
		middleware.Fail(c, cmn.StatusParamInvalid, "filehash参数无效")
		return
	}
	filename, err := util.SanitizeFileName(c.PostForm("filename"))
	if err != nil {
		middleware.Fail(c, cmn.StatusParamInvalid, err.Error())
		return
	}
	filesize, err := strconv.ParseInt(c.PostForm("filesize"), 10, 64)
	if err != nil || filesize < 0 {
		middleware.Fail(c, cmn.StatusParamInvalid, "filesize参数无效")
		return
	}

	// 2. 从文件表中查询相同hash的文件记录
	tfile, err := dbcli.GetTableFile(filehash)
	if err != nil {
		log.Println(err.Error())
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return
	}

	// 3. 查不到记录(或文件已丢失、大小不一致)则返回秒传失败
	if tfile == nil || tfile.FileSize != filesize || !fileAvailable(tfile.FileAddr, tfile.FileSize) {
		middleware.Fail(c, cmn.StatusFastUploadMiss, "秒传失败，请访问普通上传接口")
		return
	}

	// 4. 上传过则将文件信息写入用户文件表， 返回成功
	fmeta := dbcli.TableFileToFileMeta(*tfile)
	fmeta.FileName = filename
	if _, err := dbcli.OnUserFileUploadFinished(username, fmeta); err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "秒传失败，请稍后重试")
		return
	}
	middleware.OK(c, "秒传成功", gin.H{
		"FileHash": fmeta.FileSha1,
		"FileName": fmeta.FileName,
		"FileSize": fmeta.FileSize,
	})
}
