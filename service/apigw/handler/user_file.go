package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	cmn "filestore-server/common"
	"filestore-server/middleware"
	userProto "filestore-server/service/account/proto"
	"filestore-server/util"
)

// FileQueryHandler : 分页查询当前用户的文件列表
// 参数: limit(默认20, 最大100), offset(默认0)
func (h *Handler) FileQueryHandler(c *gin.Context) {
	limitCnt, _ := strconv.Atoi(c.PostForm("limit"))
	offset, _ := strconv.Atoi(c.PostForm("offset"))

	rpcResp, err := h.UserCli.UserFiles(c.Request.Context(), &userProto.ReqUserFile{
		Username: middleware.Username(c),
		Limit:    int32(limitCnt),
		Offset:   int32(offset),
	})
	if err != nil {
		rpcFailed(c, err)
		return
	}
	if rpcResp.Code != cmn.StatusOK {
		middleware.Fail(c, rpcResp.Code, rpcResp.Message)
		return
	}

	files := json.RawMessage(rpcResp.FileData)
	if len(files) == 0 {
		files = json.RawMessage("[]")
	}
	middleware.OK(c, "OK", gin.H{
		"Total": rpcResp.Total,
		"Files": files,
	})
}

// fileHashParam : 获取并校验filehash参数
func fileHashParam(c *gin.Context) (string, bool) {
	filehash := strings.ToLower(strings.TrimSpace(c.PostForm("filehash")))
	if !util.IsValidSha1(filehash) {
		middleware.Fail(c, cmn.StatusParamInvalid, "filehash参数无效")
		return "", false
	}
	return filehash, true
}

// FileMetaUpdateHandler ： 更新元信息接口(op=0: 重命名)
func (h *Handler) FileMetaUpdateHandler(c *gin.Context) {
	if c.PostForm("op") != "0" {
		middleware.Fail(c, cmn.StatusParamInvalid, "不支持的操作类型")
		return
	}
	filehash, ok := fileHashParam(c)
	if !ok {
		return
	}

	rpcResp, err := h.UserCli.UserFileRename(c.Request.Context(), &userProto.ReqUserFileRename{
		Username:    middleware.Username(c),
		Filehash:    filehash,
		NewFileName: c.PostForm("filename"),
	})
	if err != nil {
		rpcFailed(c, err)
		return
	}
	if rpcResp.Code != cmn.StatusOK {
		middleware.Fail(c, rpcResp.Code, rpcResp.Message)
		return
	}
	var data interface{}
	if len(rpcResp.FileData) > 0 {
		data = json.RawMessage(rpcResp.FileData)
	}
	middleware.Resp(c, http.StatusOK, rpcResp.Code, rpcResp.Message, data)
}

// FileDeleteHandler : 删除用户文件
func (h *Handler) FileDeleteHandler(c *gin.Context) {
	filehash, ok := fileHashParam(c)
	if !ok {
		return
	}
	rpcResp, err := h.UserCli.UserFileDelete(c.Request.Context(), &userProto.ReqUserFileDelete{
		Username: middleware.Username(c),
		Filehash: filehash,
	})
	if err != nil {
		rpcFailed(c, err)
		return
	}
	middleware.Resp(c, http.StatusOK, rpcResp.Code, rpcResp.Message, nil)
}
