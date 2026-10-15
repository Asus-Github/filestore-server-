package util

import (
	"strconv"
	"time"
)

// SignDownload : 生成下载链接签名, 签名覆盖 用户名+文件hash+过期时间
func SignDownload(secret, username, filehash string, expires int64) string {
	return HmacSha256(secret, "download\n"+username+"\n"+filehash+"\n"+strconv.FormatInt(expires, 10))
}

// VerifyDownloadSign : 校验下载链接签名及有效期
func VerifyDownloadSign(secret, username, filehash, expiresStr, sign string) (ok bool, expired bool) {
	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil || sign == "" || username == "" || filehash == "" {
		return false, false
	}
	if !SecureEqual(SignDownload(secret, username, filehash, expires), sign) {
		return false, false
	}
	if time.Now().Unix() > expires {
		return false, true
	}
	return true, false
}
