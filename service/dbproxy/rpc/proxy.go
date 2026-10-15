package rpc

import (
	"bytes"
	"context"
	"encoding/json"

	"filestore-server/service/dbproxy/mapper"
	"filestore-server/service/dbproxy/orm"
	dbProxy "filestore-server/service/dbproxy/proto"
)

// DBProxy : DBProxy结构体
type DBProxy struct{}

// ExecuteAction : 请求执行sql函数
func (db *DBProxy) ExecuteAction(ctx context.Context, req *dbProxy.ReqExec, res *dbProxy.RespExec) error {
	resList := make([]orm.ExecResult, len(req.Action))

	// TODO: 检查	req.Sequence req.Transaction两个参数，执行不同的流程
	for idx, singleAction := range req.Action {
		params := []interface{}{}
		dec := json.NewDecoder(bytes.NewReader(singleAction.Params))
		// 避免int/int32/int64等自动转换为float64
		dec.UseNumber()
		if err := dec.Decode(&params); err != nil {
			resList[idx] = orm.ExecResult{
				Suc:  false,
				Code: orm.CodeBadParam,
				Msg:  "请求参数有误",
			}
			continue
		}

		// 默认串行执行sql函数
		execRes, err := mapper.FuncCall(singleAction.Name, params...)
		if err != nil {
			resList[idx] = orm.ExecResult{
				Suc:  false,
				Code: orm.CodeBadParam,
				Msg:  "函数调用有误: " + err.Error(),
			}
			continue
		}
		resList[idx] = execRes
	}

	data, err := json.Marshal(resList)
	if err != nil {
		res.Code = -1
		res.Msg = err.Error()
		return nil
	}
	res.Data = data
	return nil
}
