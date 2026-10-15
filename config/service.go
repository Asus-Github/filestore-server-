package config

var (
	// APIGWServiceHost : 网关服务监听地址
	APIGWServiceHost = envString("FS_APIGW_ADDR", "0.0.0.0:8080")
	// UploadServiceHost : 上传服务监听的地址
	UploadServiceHost = envString("FS_UPLOAD_ADDR", "0.0.0.0:28080")
	// DownloadServiceHost : 下载服务监听的地址
	DownloadServiceHost = envString("FS_DOWNLOAD_ADDR", "0.0.0.0:38080")
	// UploadLBHost : 返回给前端的上传服务入口(LB地址)
	// 为空时前端使用"当前页面host + 上传服务端口"拼接入口, 便于本地直接访问
	UploadLBHost = envString("FS_UPLOAD_ENTRY", "")
	// DownloadLBHost : 返回给前端的下载服务入口(LB地址), 为空时规则同上
	DownloadLBHost = envString("FS_DOWNLOAD_ENTRY", "")
	// CORSAllowOrigins : 允许跨域访问上传/下载服务的来源, 逗号分隔, "*"表示全部
	CORSAllowOrigins = envString("FS_CORS_ORIGINS", "*")
	// TracerAgentHost : tracing agent地址
	TracerAgentHost = envString("FS_TRACER_AGENT", "127.0.0.1:6831")
)
