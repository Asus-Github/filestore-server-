package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go-micro.dev/v4"

	"filestore-server/service/dbproxy/orm"
	dbProto "filestore-server/service/dbproxy/proto"
)

// FileMeta : 文件元信息结构
type FileMeta struct {
	FileSha1 string
	FileName string
	FileSize int64
	Location string
	UploadAt string
}

// Result : dbproxy单个sql函数的执行结果, Data保留原始json, 由调用方按需解码为具体类型
type Result struct {
	Suc  bool            `json:"suc"`
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// NotFound : 记录不存在
func (r *Result) NotFound() bool {
	return !r.Suc && r.Code == orm.CodeNotFound
}

// Decode : 将Data解码到v
func (r *Result) Decode(v interface{}) error {
	if len(r.Data) == 0 || string(r.Data) == "null" {
		return errors.New("dbproxy: empty result data")
	}
	return json.Unmarshal(r.Data, v)
}

var (
	dbCli dbProto.DBProxyService
)

func Init(service micro.Service) {
	// 初始化一个dbproxy服务的客户端
	dbCli = dbProto.NewDBProxyService("go.micro.service.dbproxy", service.Client())
}

func TableFileToFileMeta(tfile orm.TableFile) FileMeta {
	return FileMeta{
		FileSha1: tfile.FileHash,
		FileName: tfile.FileName,
		FileSize: tfile.FileSize,
		Location: tfile.FileAddr,
	}
}

// execAction : 向dbproxy请求执行action, 保证 err==nil 时返回的结果不为nil
func execAction(funcName string, params ...interface{}) (*Result, error) {
	paramJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	resp, err := dbCli.ExecuteAction(context.Background(), &dbProto.ReqExec{
		Action: []*dbProto.SingleAction{
			{
				Name:   funcName,
				Params: paramJSON,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dbproxy %s: %w", funcName, err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("dbproxy %s: %s", funcName, resp.Msg)
	}
	resList := []Result{}
	if err := json.Unmarshal(resp.Data, &resList); err != nil {
		return nil, fmt.Errorf("dbproxy %s: invalid response: %w", funcName, err)
	}
	if len(resList) == 0 {
		return nil, fmt.Errorf("dbproxy %s: empty response", funcName)
	}
	res := &resList[0]
	if !res.Suc && res.Code == orm.CodeBadParam {
		// 参数/函数名错误属于程序bug, 作为error返回便于排查
		return nil, fmt.Errorf("dbproxy %s: %s", funcName, res.Msg)
	}
	return res, nil
}

// execExpect : 执行action, 失败(包括记录不存在)时返回error
func execExpect(funcName string, params ...interface{}) (*Result, error) {
	res, err := execAction(funcName, params...)
	if err != nil {
		return nil, err
	}
	if !res.Suc {
		return res, fmt.Errorf("dbproxy %s: %s", funcName, res.Msg)
	}
	return res, nil
}

func GetFileMeta(filehash string) (*Result, error) {
	return execAction("/file/GetFileMeta", filehash)
}

// GetTableFile : 查询唯一文件表记录, 不存在时返回(nil, nil)
func GetTableFile(filehash string) (*orm.TableFile, error) {
	res, err := GetFileMeta(filehash)
	if err != nil {
		return nil, err
	}
	if res.NotFound() {
		return nil, nil
	}
	if !res.Suc {
		return nil, errors.New(res.Msg)
	}
	tfile := orm.TableFile{}
	if err := res.Decode(&tfile); err != nil {
		return nil, err
	}
	return &tfile, nil
}

func GetFileMetaList(limitCnt int) (*Result, error) {
	return execAction("/file/GetFileMetaList", limitCnt)
}

// OnFileUploadFinished : 新增/更新文件元信息到mysql中
func OnFileUploadFinished(fmeta FileMeta) (*Result, error) {
	return execExpect("/file/OnFileUploadFinished",
		fmeta.FileSha1, fmeta.FileName, fmeta.FileSize, fmeta.Location)
}

func UpdateFileLocation(filehash, location string) (*Result, error) {
	return execAction("/file/UpdateFileLocation", filehash, location)
}

func UserSignup(username, encPasswd string) (*Result, error) {
	return execAction("/user/UserSignup", username, encPasswd)
}

// GetUserAuth : 查询用户认证信息, 用户不存在时返回(nil, nil)
func GetUserAuth(username string) (*orm.TableUserAuth, error) {
	res, err := execAction("/user/GetUserAuth", username)
	if err != nil {
		return nil, err
	}
	if res.NotFound() {
		return nil, nil
	}
	if !res.Suc {
		return nil, errors.New(res.Msg)
	}
	auth := orm.TableUserAuth{}
	if err := res.Decode(&auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

func UpdateUserPassword(username, encPasswd string) (*Result, error) {
	return execExpect("/user/UpdateUserPassword", username, encPasswd)
}

func CreateToken(username, tokenHash string, expireAt int64) (*Result, error) {
	return execExpect("/user/CreateToken", username, tokenHash, expireAt)
}

// GetTokenUser : 查询token所属用户, token无效时返回("", nil)
func GetTokenUser(tokenHash string) (string, error) {
	res, err := execAction("/user/GetTokenUser", tokenHash)
	if err != nil {
		return "", err
	}
	if res.NotFound() {
		return "", nil
	}
	if !res.Suc {
		return "", errors.New(res.Msg)
	}
	var username string
	if err := res.Decode(&username); err != nil {
		return "", err
	}
	return username, nil
}

func DeleteToken(tokenHash string) (*Result, error) {
	return execExpect("/user/DeleteToken", tokenHash)
}

func GetUserInfo(username string) (*Result, error) {
	return execAction("/user/GetUserInfo", username)
}

func UserExist(username string) (*Result, error) {
	return execAction("/user/UserExist", username)
}

func QueryUserFileMeta(username, filehash string) (*Result, error) {
	return execAction("/ufile/QueryUserFileMeta", username, filehash)
}

// GetUserFile : 查询用户文件记录, 不存在时返回(nil, nil)
func GetUserFile(username, filehash string) (*orm.TableUserFile, error) {
	res, err := QueryUserFileMeta(username, filehash)
	if err != nil {
		return nil, err
	}
	if res.NotFound() {
		return nil, nil
	}
	if !res.Suc {
		return nil, errors.New(res.Msg)
	}
	ufile := orm.TableUserFile{}
	if err := res.Decode(&ufile); err != nil {
		return nil, err
	}
	return &ufile, nil
}

func QueryUserFileMetas(username string, limit, offset int) (*Result, error) {
	return execAction("/ufile/QueryUserFileMetas", username, limit, offset)
}

// OnUserFileUploadFinished : 新增/更新文件元信息到mysql中
func OnUserFileUploadFinished(username string, fmeta FileMeta) (*Result, error) {
	return execExpect("/ufile/OnUserFileUploadFinished", username, fmeta.FileSha1,
		fmeta.FileName, fmeta.FileSize)
}

func RenameFileName(username, filehash, filename string) (*Result, error) {
	return execAction("/ufile/RenameFileName", username, filehash, filename)
}

func DeleteUserFile(username, filehash string) (*Result, error) {
	return execAction("/ufile/DeleteUserFile", username, filehash)
}
