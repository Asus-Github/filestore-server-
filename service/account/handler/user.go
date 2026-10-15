package handler

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"filestore-server/common"
	cfg "filestore-server/config"
	proto "filestore-server/service/account/proto"
	dbcli "filestore-server/service/dbproxy/client"
	"filestore-server/service/dbproxy/orm"
	"filestore-server/util"
)

const (
	// 密码长度限制(bcrypt最多只使用前72字节)
	minPasswordLen = 6
	maxPasswordLen = 64
	// tokenBytes : token随机字节数, 对应64位hex字符
	tokenBytes = 32
)

// User : 用于实现UserServiceHandler接口的对象
type User struct{}

// GenToken : 生成随机的访问token(64位hex字符), 不可预测
func GenToken() (string, error) {
	return util.RandomHex(tokenBytes)
}

// HashToken : 数据库中只保存token的sha256, 即使数据泄露也无法直接冒用
func HashToken(token string) string {
	return util.Sha256([]byte(token))
}

// IsTokenFormatValid : token格式校验
func IsTokenFormatValid(token string) bool {
	if len(token) != tokenBytes*2 {
		return false
	}
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// encryptPassword : 使用bcrypt加密密码
func encryptPassword(passwd string) (string, error) {
	enc, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	return string(enc), err
}

// legacyEncrypt : 旧版本的密码加密方式(sha1+全局盐), 仅用于兼容历史数据
func legacyEncrypt(passwd string) string {
	return util.Sha1([]byte(passwd + cfg.PasswordSalt))
}

// checkPassword : 校验密码, 返回(是否匹配, 是否需要升级为bcrypt)
func checkPassword(stored, passwd string) (bool, bool) {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(passwd)) == nil, false
	}
	if len(stored) == 40 && util.SecureEqual(stored, legacyEncrypt(passwd)) {
		return true, true
	}
	return false, false
}

func validPassword(passwd string) bool {
	n := utf8.RuneCountInString(passwd)
	return n >= minPasswordLen && n <= maxPasswordLen && len(passwd) <= 72
}

// Signup : 处理用户注册请求
func (u *User) Signup(ctx context.Context, req *proto.ReqSignup, res *proto.RespSignup) error {
	username := strings.TrimSpace(req.Username)
	passwd := req.Password

	// 参数校验
	if !util.IsValidUsername(username) {
		res.Code = common.StatusParamInvalid
		res.Message = "用户名需为3~32位字母、数字、下划线、点或中划线"
		return nil
	}
	if !validPassword(passwd) {
		res.Code = common.StatusParamInvalid
		res.Message = "密码长度需为6~64位"
		return nil
	}

	encPasswd, err := encryptPassword(passwd)
	if err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	// 将用户信息注册到用户表中
	dbResp, err := dbcli.UserSignup(username, encPasswd)
	switch {
	case err != nil:
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误, 请稍后重试"
	case dbResp.Suc:
		res.Code = common.StatusOK
		res.Message = "注册成功"
	case dbResp.Code == orm.CodeDuplicate:
		res.Code = common.StatusUserExists
		res.Message = "用户名已被注册"
	default:
		res.Code = common.StatusRegisterFailed
		res.Message = "注册失败"
	}
	return nil
}

// Signin : 处理用户登录请求
func (u *User) Signin(ctx context.Context, req *proto.ReqSignin, res *proto.RespSignin) error {
	username := strings.TrimSpace(req.Username)
	password := req.Password
	if username == "" || password == "" {
		res.Code = common.StatusParamInvalid
		res.Message = "请输入用户名和密码"
		return nil
	}

	// 1. 校验用户名及密码
	auth, err := dbcli.GetUserAuth(username)
	if err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误, 请稍后重试"
		return nil
	}
	if auth == nil {
		// 用户不存在时也进行一次bcrypt计算, 避免通过响应时间探测用户名是否存在
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		res.Code = common.StatusLoginFailed
		res.Message = "用户名或密码错误"
		return nil
	}
	matched, needUpgrade := checkPassword(auth.Password, password)
	if !matched {
		res.Code = common.StatusLoginFailed
		res.Message = "用户名或密码错误"
		return nil
	}
	if auth.Status != orm.UserStatusNormal {
		res.Code = common.StatusLoginFailed
		res.Message = "账号已被禁用"
		return nil
	}
	if needUpgrade {
		// 历史sha1密码在登录成功后自动升级为bcrypt
		if enc, err := encryptPassword(password); err == nil {
			if _, err := dbcli.UpdateUserPassword(auth.Username, enc); err != nil {
				log.Println(err)
			}
		}
	}

	// 2. 生成访问凭证(token)
	token, err := GenToken()
	if err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	expireAt := time.Now().Add(cfg.TokenTTL).Unix()
	if _, err := dbcli.CreateToken(auth.Username, HashToken(token), expireAt); err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误, 请稍后重试"
		return nil
	}

	// 3. 登录成功, 返回token
	res.Code = common.StatusOK
	res.Message = "登录成功"
	res.Token = token
	res.ExpireAt = expireAt
	res.Username = auth.Username
	return nil
}

// dummyHash : 用于用户不存在时的等时比较
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("filestore-dummy-password"), bcrypt.DefaultCost)

// Signout : 退出登录, 注销token
func (u *User) Signout(ctx context.Context, req *proto.ReqSignout, res *proto.RespSignout) error {
	if !IsTokenFormatValid(req.Token) {
		res.Code = common.StatusOK
		res.Message = "OK"
		return nil
	}
	if _, err := dbcli.DeleteToken(HashToken(req.Token)); err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	res.Code = common.StatusOK
	res.Message = "OK"
	return nil
}

// VerifyToken : 校验token是否有效(存在、未过期、归属用户一致)
func (u *User) VerifyToken(ctx context.Context, req *proto.ReqVerifyToken, res *proto.RespVerifyToken) error {
	if !IsTokenFormatValid(req.Token) {
		res.Code = common.StatusTokenInvalid
		res.Message = "token无效"
		return nil
	}
	username, err := dbcli.GetTokenUser(HashToken(req.Token))
	if err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	if username == "" || (req.Username != "" && req.Username != username) {
		res.Code = common.StatusTokenInvalid
		res.Message = "登录已失效, 请重新登录"
		return nil
	}
	res.Code = common.StatusOK
	res.Message = "OK"
	res.Username = username
	return nil
}

// UserInfo ： 查询用户信息
func (u *User) UserInfo(ctx context.Context, req *proto.ReqUserInfo, res *proto.RespUserInfo) error {
	// 查询用户信息
	dbResp, err := dbcli.GetUserInfo(req.Username)
	if err != nil {
		log.Println(err)
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}
	// 查不到对应的用户信息
	if dbResp.NotFound() {
		res.Code = common.StatusUserNotExists
		res.Message = "用户不存在"
		return nil
	}
	user := orm.TableUser{}
	if !dbResp.Suc || dbResp.Decode(&user) != nil {
		res.Code = common.StatusServerError
		res.Message = "服务错误"
		return nil
	}

	// 组装并且响应用户数据
	res.Code = common.StatusOK
	res.Message = "OK"
	res.Username = user.Username
	res.SignupAt = user.SignupAt
	res.LastActiveAt = user.LastActiveAt
	res.Status = int32(user.Status)
	res.Email = user.Email
	res.Phone = user.Phone
	return nil
}
