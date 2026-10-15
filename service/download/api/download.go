package api

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	cmn "filestore-server/common"
	cfg "filestore-server/config"
	"filestore-server/middleware"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/service/dbproxy/orm"
	"filestore-server/store/ceph"
	"filestore-server/store/oss"
	"filestore-server/util"
)

// userFileInfo : 校验文件归属当前用户, 并查询唯一文件表记录; 失败时已输出响应
func userFileInfo(c *gin.Context, username, filehash string) (*orm.TableUserFile, *orm.TableFile, bool) {
	if !util.IsValidSha1(filehash) {
		middleware.Fail(c, cmn.StatusParamInvalid, "filehash参数无效")
		return nil, nil, false
	}
	userFile, err := dbcli.GetUserFile(username, filehash)
	if err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return nil, nil, false
	}
	if userFile == nil {
		middleware.Fail(c, cmn.StatusFileNotExists, "文件不存在")
		return nil, nil, false
	}
	uniqFile, err := dbcli.GetTableFile(filehash)
	if err != nil {
		log.Println(err)
		middleware.Fail(c, cmn.StatusServerError, "服务错误")
		return nil, nil, false
	}
	if uniqFile == nil {
		middleware.Fail(c, cmn.StatusFileNotExists, "文件不存在")
		return nil, nil, false
	}
	return userFile, uniqFile, true
}

// DownloadURLHandler : 生成文件的下载地址(需登录)
// 本地/Ceph文件返回带签名的临时下载链接, 不在url中暴露用户token; OSS文件返回OSS的签名url
func DownloadURLHandler(c *gin.Context) {
	username := middleware.Username(c)
	filehash := strings.ToLower(strings.TrimSpace(c.PostForm("filehash")))
	userFile, uniqFile, ok := userFileInfo(c, username, filehash)
	if !ok {
		return
	}

	addr := uniqFile.FileAddr
	expires := time.Now().Add(cfg.DownloadURLTTL).Unix()
	var dlURL string
	switch {
	case strings.HasPrefix(addr, cfg.TempLocalRootDir) || strings.HasPrefix(addr, cfg.CephRootDir):
		q := url.Values{}
		q.Set("filehash", filehash)
		q.Set("username", username)
		q.Set("expires", strconv.FormatInt(expires, 10))
		q.Set("sign", util.SignDownload(cfg.TokenSecret, username, filehash, expires))
		dlURL = middleware.RequestBaseURL(c) + "/file/download?" + q.Encode()
	case strings.HasPrefix(addr, cfg.OSSRootDir):
		signedURL, err := oss.DownloadURL(addr, userFile.FileName, int64(cfg.DownloadURLTTL.Seconds()))
		if err != nil {
			log.Println(err)
			middleware.Fail(c, cmn.StatusServerError, "生成下载链接失败")
			return
		}
		dlURL = signedURL
	default:
		log.Printf("unknown file location: %s", addr)
		middleware.Fail(c, cmn.StatusFileNotExists, "文件不存在")
		return
	}

	middleware.OK(c, "OK", gin.H{
		"URL":      dlURL,
		"FileName": userFile.FileName,
		"FileSize": userFile.FileSize,
		"ExpireAt": expires,
	})
}

// DownloadHandler : 文件下载接口, 支持Range请求(断点续传/分段下载)
// 鉴权方式: 1) 签名链接 filehash+username+expires+sign; 2) 登录token(Authorization头)
func DownloadHandler(c *gin.Context, verify func(c *gin.Context) (string, bool)) {
	filehash := strings.ToLower(strings.TrimSpace(c.Query("filehash")))

	var username string
	if sign := c.Query("sign"); sign != "" {
		ok, expired := util.VerifyDownloadSign(cfg.TokenSecret, c.Query("username"), filehash, c.Query("expires"), sign)
		if expired {
			middleware.AbortWith(c, http.StatusForbidden, cmn.StatusURLExpired, "下载链接已失效, 请重新获取")
			return
		}
		if !ok {
			middleware.AbortWith(c, http.StatusForbidden, cmn.StatusForbidden, "下载链接无效")
			return
		}
		username = c.Query("username")
	} else {
		var ok bool
		if username, ok = verify(c); !ok {
			return
		}
	}

	userFile, uniqFile, ok := userFileInfo(c, username, filehash)
	if !ok {
		return
	}
	addr := uniqFile.FileAddr
	disposition := "attachment"
	if c.Query("inline") == "1" {
		disposition = "inline"
	}
	c.Header("Content-Disposition", util.ContentDisposition(disposition, userFile.FileName))
	c.Header("Cache-Control", "private, max-age=0")

	switch {
	case strings.HasPrefix(addr, cfg.TempLocalRootDir):
		// 本地文件, http.ServeContent 会处理 Range/If-Range/HEAD, 返回206及Content-Range
		f, err := os.Open(addr)
		if err != nil {
			log.Println(err)
			middleware.Fail(c, cmn.StatusFileNotExists, "文件不存在或已损坏")
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			middleware.Fail(c, cmn.StatusServerError, "服务错误")
			return
		}
		c.Header("Content-Type", "application/octet-stream")
		c.Header("ETag", `"`+filehash+`"`)
		http.ServeContent(c.Writer, c.Request, "", fi.ModTime(), f)
	case strings.HasPrefix(addr, cfg.CephRootDir):
		// ceph中的文件，通过ceph api先下载
		data, err := ceph.GetCephBucket(cfg.CephBucket).Get(addr)
		if err != nil {
			log.Println(err)
			middleware.Fail(c, cmn.StatusServerError, "读取文件失败")
			return
		}
		c.Header("Content-Type", "application/octet-stream")
		c.Header("ETag", `"`+filehash+`"`)
		http.ServeContent(c.Writer, c.Request, "", time.Time{}, bytes.NewReader(data))
	case strings.HasPrefix(addr, cfg.OSSRootDir):
		signedURL, err := oss.DownloadURL(addr, userFile.FileName, int64(cfg.DownloadURLTTL.Seconds()))
		if err != nil {
			log.Println(err)
			middleware.Fail(c, cmn.StatusServerError, "生成下载链接失败")
			return
		}
		c.Redirect(http.StatusFound, signedURL)
	default:
		middleware.Fail(c, cmn.StatusFileNotExists, "文件不存在")
	}
}
