// Package plugins : 注册go-micro的registry插件, 使服务可通过 --registry=consul|etcd|kubernetes 切换注册中心
// (默认使用mdns, 本地开发无需额外部署注册中心)
package plugins

import (
	_ "github.com/go-micro/plugins/v4/registry/consul"
	_ "github.com/go-micro/plugins/v4/registry/etcd"
	_ "github.com/go-micro/plugins/v4/registry/kubernetes"
)
