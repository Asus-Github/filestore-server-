package config

var (
	// OSSBucket : oss bucket名
	OSSBucket = envString("FS_OSS_BUCKET", "buckettest-filestore2")
	// OSSEndpoint : oss endpoint
	OSSEndpoint = envString("FS_OSS_ENDPOINT", "oss-cn-shenzhen.aliyuncs.com")
	// OSSAccesskeyID : oss访问key(请通过环境变量注入, 不要写死在代码中)
	OSSAccesskeyID = envString("FS_OSS_ACCESS_KEY_ID", "")
	// OSSAccessKeySecret : oss访问key secret
	OSSAccessKeySecret = envString("FS_OSS_ACCESS_KEY_SECRET", "")
)
