package mapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"filestore-server/service/dbproxy/orm"
)

var funcs = map[string]interface{}{
	"/file/OnFileUploadFinished": orm.OnFileUploadFinished,
	"/file/GetFileMeta":          orm.GetFileMeta,
	"/file/GetFileMetaList":      orm.GetFileMetaList,
	"/file/UpdateFileLocation":   orm.UpdateFileLocation,

	"/user/UserSignup":         orm.UserSignup,
	"/user/GetUserAuth":        orm.GetUserAuth,
	"/user/UpdateUserPassword": orm.UpdateUserPassword,
	"/user/CreateToken":        orm.CreateToken,
	"/user/GetTokenUser":       orm.GetTokenUser,
	"/user/DeleteToken":        orm.DeleteToken,
	"/user/GetUserInfo":        orm.GetUserInfo,
	"/user/UserExist":          orm.UserExist,

	"/ufile/OnUserFileUploadFinished": orm.OnUserFileUploadFinished,
	"/ufile/QueryUserFileMetas":       orm.QueryUserFileMetas,
	"/ufile/DeleteUserFile":           orm.DeleteUserFile,
	"/ufile/RenameFileName":           orm.RenameFileName,
	"/ufile/QueryUserFileMeta":        orm.QueryUserFileMeta,
}

var execResultType = reflect.TypeOf(orm.ExecResult{})

func init() {
	// 启动时校验注册的函数签名, 避免运行期调用时panic
	for name, f := range funcs {
		t := reflect.TypeOf(f)
		if t.Kind() != reflect.Func || t.NumOut() != 1 || t.Out(0) != execResultType {
			panic(fmt.Sprintf("mapper: %s must be func(...) orm.ExecResult", name))
		}
	}
}

// FuncCall : 按函数名调用orm函数, params 为json解码后的参数(数字需为json.Number)
func FuncCall(name string, params ...interface{}) (result orm.ExecResult, err error) {
	fn, ok := funcs[name]
	if !ok {
		err = errors.New("函数名不存在: " + name)
		return
	}

	// 通过反射可以动态调用对象的导出方法
	f := reflect.ValueOf(fn)
	ft := f.Type()
	if len(params) != ft.NumIn() {
		err = fmt.Errorf("%s: 传入参数数量(%d)与被调用方法要求的数量(%d)不一致", name, len(params), ft.NumIn())
		return
	}

	// 构造一个 Value的slice, 用作Call()方法的传入参数; 参数类型不匹配时返回错误而不是panic
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		v, cerr := convertParam(param, ft.In(k))
		if cerr != nil {
			err = fmt.Errorf("%s: 第%d个参数有误: %v", name, k+1, cerr)
			return
		}
		in[k] = v
	}

	// 执行方法f, 并将方法结果赋值给result
	result = f.Call(in)[0].Interface().(orm.ExecResult)
	return
}

// convertParam : 将json参数转换为目标类型
func convertParam(param interface{}, target reflect.Type) (reflect.Value, error) {
	switch target.Kind() {
	case reflect.String:
		if s, ok := param.(string); ok {
			return reflect.ValueOf(s).Convert(target), nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, ok := param.(json.Number); ok {
			i, err := n.Int64()
			if err != nil {
				return reflect.Value{}, err
			}
			v := reflect.New(target).Elem()
			if v.OverflowInt(i) {
				return reflect.Value{}, fmt.Errorf("数值溢出: %d", i)
			}
			v.SetInt(i)
			return v, nil
		}
	case reflect.Bool:
		if b, ok := param.(bool); ok {
			return reflect.ValueOf(b), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("期望类型 %s, 实际为 %T", target, param)
}
