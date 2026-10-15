package config

var (
	// CephAccessKey : 访问Key(请通过环境变量注入, 不要写死在代码中)
	CephAccessKey = envString("FS_CEPH_ACCESS_KEY", "")
	// CephSecretKey : 访问密钥
	CephSecretKey = envString("FS_CEPH_SECRET_KEY", "")
	// CephGWEndpoint : gateway地址, 如 http://<rgw_host>:<rgw_port>
	CephGWEndpoint = envString("FS_CEPH_ENDPOINT", "http://127.0.0.1:7480")
	// CephBucket : 存放用户文件的bucket
	CephBucket = envString("FS_CEPH_BUCKET", "userfile")
)
