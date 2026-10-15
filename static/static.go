// Package static : 前端静态资源, 编译时通过 go:embed 打包进网关可执行文件
package static

import "embed"

// FS : 静态资源文件系统, 路径形如 view/home.html、js/app.js
//
//go:embed view css js img
var FS embed.FS
