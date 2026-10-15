package oss

import (
	"errors"
	"log"
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	cfg "filestore-server/config"
	"filestore-server/util"
)

var (
	ossCli  *oss.Client
	ossErr  error
	cliOnce sync.Once
)

// ErrUnavailable : oss客户端不可用(配置缺失等)
var ErrUnavailable = errors.New("oss client is unavailable, please check oss config")

// Client : 创建oss client对象(只创建一次)
func Client() *oss.Client {
	cliOnce.Do(func() {
		ossCli, ossErr = oss.New(cfg.OSSEndpoint, cfg.OSSAccesskeyID, cfg.OSSAccessKeySecret)
		if ossErr != nil {
			log.Println("create oss client failed:", ossErr)
			ossCli = nil
		}
	})
	return ossCli
}

// Bucket : 获取bucket存储空间
func Bucket() (*oss.Bucket, error) {
	cli := Client()
	if cli == nil {
		return nil, ErrUnavailable
	}
	return cli.Bucket(cfg.OSSBucket)
}

// DownloadURL : 临时授权下载url, filename用于指定下载时保存的文件名
func DownloadURL(objName string, filename string, expireSeconds int64) (string, error) {
	bucket, err := Bucket()
	if err != nil {
		return "", err
	}
	opts := []oss.Option{}
	if filename != "" {
		opts = append(opts, oss.ResponseContentDisposition(util.ContentDisposition("attachment", filename)))
	}
	return bucket.SignURL(objName, oss.HTTPGet, expireSeconds, opts...)
}

// BuildLifecycleRule : 针对指定bucket设置生命周期规则
func BuildLifecycleRule(bucketName string) error {
	cli := Client()
	if cli == nil {
		return ErrUnavailable
	}
	// 表示前缀为test的对象(文件)距最后修改时间30天后过期。
	ruleTest1 := oss.BuildLifecycleRuleByDays("rule1", "test/", true, 30)
	rules := []oss.LifecycleRule{ruleTest1}

	return cli.SetBucketLifecycle(bucketName, rules)
}

// GenFileMeta :  构造文件元信息
func GenFileMeta(metas map[string]string) []oss.Option {
	options := []oss.Option{}
	for k, v := range metas {
		options = append(options, oss.Meta(k, v))
	}
	return options
}
