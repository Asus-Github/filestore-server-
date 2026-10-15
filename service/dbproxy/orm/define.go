package orm

// 用户文件状态(tbl_user_file.status)
const (
	// UserFileStatusNormal : 正常
	UserFileStatusNormal = 0
	// UserFileStatusDeleted : 已删除(标记删除)
	UserFileStatusDeleted = 1
	// UserFileStatusDisabled : 禁用
	UserFileStatusDisabled = 2
)

// 唯一文件状态(tbl_file.status)
const (
	// FileStatusAvailable : 可用
	FileStatusAvailable = 1
)

// 用户状态(tbl_user.status)
const (
	// UserStatusNormal : 正常
	UserStatusNormal = 0
)

// ExecResult.Code 的取值
const (
	// CodeOK : 执行成功
	CodeOK = 0
	// CodeError : 执行出错(sql错误等)
	CodeError = -1
	// CodeNotFound : 记录不存在
	CodeNotFound = 1
	// CodeDuplicate : 记录已存在(唯一键冲突)
	CodeDuplicate = 2
	// CodeBadParam : 参数有误
	CodeBadParam = 3
)

// TableFile : 文件表结构体
type TableFile struct {
	FileHash string `json:"FileHash"`
	FileName string `json:"FileName"`
	FileSize int64  `json:"FileSize"`
	FileAddr string `json:"FileAddr"`
}

// TableUser : 用户表model
type TableUser struct {
	Username     string `json:"Username"`
	Email        string `json:"Email"`
	Phone        string `json:"Phone"`
	SignupAt     string `json:"SignupAt"`
	LastActiveAt string `json:"LastActiveAt"`
	Status       int    `json:"Status"`
}

// TableUserAuth : 用户认证信息(仅在服务内部使用, 不返回给前端)
type TableUserAuth struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
	Status   int    `json:"Status"`
}

// TableUserFile : 用户文件表结构体
type TableUserFile struct {
	UserName    string `json:"UserName"`
	FileHash    string `json:"FileHash"`
	FileName    string `json:"FileName"`
	FileSize    int64  `json:"FileSize"`
	UploadAt    string `json:"UploadAt"`
	LastUpdated string `json:"LastUpdated"`
}

// UserFilePage : 分页查询的用户文件列表
type UserFilePage struct {
	Total int64           `json:"Total"`
	Files []TableUserFile `json:"Files"`
}

// ExecResult: sql函数执行的结果
type ExecResult struct {
	Suc  bool        `json:"suc"`
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func okResult(data interface{}) ExecResult {
	return ExecResult{Suc: true, Code: CodeOK, Data: data}
}

func errResult(err error) ExecResult {
	return ExecResult{Suc: false, Code: CodeError, Msg: err.Error()}
}

func notFoundResult(msg string) ExecResult {
	return ExecResult{Suc: false, Code: CodeNotFound, Msg: msg}
}
