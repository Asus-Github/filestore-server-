package ceph

import (
	"sync"

	"gopkg.in/amz.v1/aws"
	"gopkg.in/amz.v1/s3"

	cfg "filestore-server/config"
)

var (
	cephConn *s3.S3
	connOnce sync.Once
)

// GetCephConnection : 获取ceph连接(只创建一次)
func GetCephConnection() *s3.S3 {
	connOnce.Do(func() {
		// 1. 初始化ceph的一些信息
		auth := aws.Auth{
			AccessKey: cfg.CephAccessKey,
			SecretKey: cfg.CephSecretKey,
		}

		curRegion := aws.Region{
			Name:                 "default",
			EC2Endpoint:          cfg.CephGWEndpoint,
			S3Endpoint:           cfg.CephGWEndpoint,
			S3BucketEndpoint:     "",
			S3LocationConstraint: false,
			S3LowercaseBucket:    false,
			Sign:                 aws.SignV2,
		}

		// 2. 创建S3类型的连接
		cephConn = s3.New(auth, curRegion)
	})
	return cephConn
}

// GetCephBucket : 获取指定的bucket对象
func GetCephBucket(bucket string) *s3.Bucket {
	conn := GetCephConnection()
	return conn.Bucket(bucket)
}

// PutObject : 上传文件到ceph集群(私有读写, 下载需经过下载服务鉴权)
func PutObject(bucket string, path string, data []byte) error {
	return GetCephBucket(bucket).Put(path, data, "application/octet-stream", s3.Private)
}
