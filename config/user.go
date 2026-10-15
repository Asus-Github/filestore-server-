package config

import "time"

// DefaultTokenSecret : 开发环境默认的签名密钥, 仅用于本地调试
const DefaultTokenSecret = "filestore-dev-token-secret"

var (
	// PasswordSalt : 旧版sha1密码使用的全局盐(仅用于兼容校验历史数据, 新密码统一使用bcrypt)
	PasswordSalt = envString("FS_PASSWORD_SALT", "*#890")
	// TokenSecret : 生成/校验token签名的密钥(FS_TOKEN_SECRET), 生产环境务必修改
	TokenSecret = envString("FS_TOKEN_SECRET", DefaultTokenSecret)
	// TokenTTL : 登录token有效期
	TokenTTL = envDuration("FS_TOKEN_TTL", 24*time.Hour)
	// DownloadURLTTL : 下载临时链接有效期
	DownloadURLTTL = envDuration("FS_DOWNLOAD_URL_TTL", time.Hour)
)
