package common

// ErrorCode : 错误码
type ErrorCode int32

const (
	_ int32 = iota + 9999
	// StatusOK : 10000 正常
	StatusOK
	// StatusParamInvalid : 10001 请求参数无效
	StatusParamInvalid
	// StatusServerError : 10002 服务出错
	StatusServerError
	// StatusRegisterFailed : 10003 注册失败
	StatusRegisterFailed
	// StatusLoginFailed : 10004 登录失败
	StatusLoginFailed
	// StatusTokenInvalid : 10005 token无效
	StatusTokenInvalid
	// StatusUserNotExists : 10006 用户不存在
	StatusUserNotExists
	// StatusUserExists : 10007 用户名已被注册
	StatusUserExists
	// StatusFileNotExists : 10008 文件不存在
	StatusFileNotExists
	// StatusFastUploadMiss : 10009 秒传未命中(服务端无此文件, 需走普通/分块上传)
	StatusFastUploadMiss
	// StatusUploadFailed : 10010 上传失败
	StatusUploadFailed
	// StatusChunkIncomplete : 10011 分块未全部上传
	StatusChunkIncomplete
	// StatusFileVerifyFailed : 10012 文件校验失败(hash/大小不一致)
	StatusFileVerifyFailed
	// StatusForbidden : 10013 无权访问
	StatusForbidden
	// StatusURLExpired : 10014 下载链接已失效
	StatusURLExpired
	// StatusFileTooLarge : 10015 文件过大
	StatusFileTooLarge
)
