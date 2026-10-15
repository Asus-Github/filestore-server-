package handler

import (
	"context"
	"encoding/json"
	"log"

	"filestore-server/common"
	proto "filestore-server/service/account/proto"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/service/dbproxy/orm"
	"filestore-server/util"
)

const (
	defaultFileLimit = 20
	maxFileLimit     = 100
)

// UserFiles : 分页获取用户文件列表
func (u *User) UserFiles(ctx context.Context, req *proto.ReqUserFile, res *proto.RespUserFile) error {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = defaultFileLimit
	} else if limit > maxFileLimit {
		limit = maxFileLimit
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	dbResp, err := dbcli.QueryUserFileMetas(req.Username, limit, offset)
	page := orm.UserFilePage{}
	if err != nil || !dbResp.Suc || dbResp.Decode(&page) != nil {
		log.Println("query user files failed:", err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	if page.Files == nil {
		page.Files = []orm.TableUserFile{}
	}

	data, err := json.Marshal(page.Files)
	if err != nil {
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}

	res.Code = common.StatusOK
	res.Message = "OK"
	res.FileData = data
	res.Total = page.Total
	return nil
}

// UserFileRename : 用户文件重命名
func (u *User) UserFileRename(ctx context.Context, req *proto.ReqUserFileRename, res *proto.RespUserFileRename) error {
	newName, err := util.ValidateNewFileName(req.NewFileName)
	if err != nil {
		res.Code = common.StatusParamInvalid
		res.Message = err.Error()
		return nil
	}

	dbResp, err := dbcli.RenameFileName(req.Username, req.Filehash, newName)
	if err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	if dbResp.NotFound() {
		res.Code = common.StatusFileNotExists
		res.Message = "文件不存在"
		return nil
	}
	if !dbResp.Suc {
		res.Code = common.StatusServerError
		res.Message = "重命名失败"
		return nil
	}

	res.Code = common.StatusOK
	res.Message = "重命名成功"
	res.FileData = dbResp.Data
	return nil
}

// UserFileDelete : 删除用户文件(标记删除, 不影响其他用户秒传得到的同一文件)
func (u *User) UserFileDelete(ctx context.Context, req *proto.ReqUserFileDelete, res *proto.RespUserFileDelete) error {
	dbResp, err := dbcli.DeleteUserFile(req.Username, req.Filehash)
	if err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	if dbResp.NotFound() {
		res.Code = common.StatusFileNotExists
		res.Message = "文件不存在"
		return nil
	}
	if !dbResp.Suc {
		res.Code = common.StatusServerError
		res.Message = "删除失败"
		return nil
	}
	res.Code = common.StatusOK
	res.Message = "删除成功"
	return nil
}
