package orm

import (
	"database/sql"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"

	mydb "filestore-server/service/dbproxy/conn"
)

// mysqlErrDuplicate : 唯一键冲突的错误码
const mysqlErrDuplicate = 1062

func isDuplicateErr(err error) bool {
	if me, ok := err.(*mysql.MySQLError); ok {
		return me.Number == mysqlErrDuplicate
	}
	return false
}

// UserSignup : 通过用户名及密码(已加密)完成user表的注册操作
func UserSignup(username string, passwd string) (res ExecResult) {
	_, err := mydb.DBConn().Exec(
		"insert into tbl_user (`user_name`,`user_pwd`,`status`) values (?,?,?)",
		username, passwd, UserStatusNormal)
	if err != nil {
		if isDuplicateErr(err) {
			return ExecResult{Suc: false, Code: CodeDuplicate, Msg: "用户名已被注册"}
		}
		log.Println("Failed to insert user, err:" + err.Error())
		return errResult(err)
	}
	return okResult(nil)
}

// GetUserAuth : 查询用户的认证信息(加密后的密码及状态), 由account服务校验密码
func GetUserAuth(username string) (res ExecResult) {
	auth := TableUserAuth{}
	err := mydb.DBConn().QueryRow(
		"select user_name,user_pwd,status from tbl_user where user_name=? limit 1",
		username).Scan(&auth.Username, &auth.Password, &auth.Status)
	if err == sql.ErrNoRows {
		return notFoundResult("用户名未注册")
	} else if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(auth)
}

// UpdateUserPassword : 更新用户密码(已加密), 用于旧版sha1密码升级为bcrypt
func UpdateUserPassword(username string, passwd string) (res ExecResult) {
	_, err := mydb.DBConn().Exec(
		"update tbl_user set user_pwd=? where user_name=? limit 1", passwd, username)
	if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(nil)
}

// CreateToken : 保存用户登录的token(只存储token的sha256), 同时清理该用户已过期的token
func CreateToken(username string, tokenHash string, expireAt int64) (res ExecResult) {
	conn := mydb.DBConn()
	if _, err := conn.Exec(
		"delete from tbl_user_token where user_name=? and expire_at<=NOW()", username); err != nil {
		log.Println(err.Error())
	}
	_, err := conn.Exec(
		"insert into tbl_user_token (`user_name`,`token_hash`,`expire_at`) values (?,?,?)",
		username, tokenHash, time.Unix(expireAt, 0).Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	if _, err := conn.Exec(
		"update tbl_user set last_active=NOW() where user_name=? limit 1", username); err != nil {
		log.Println(err.Error())
	}
	return okResult(nil)
}

// GetTokenUser : 查询未过期token所属的用户名
func GetTokenUser(tokenHash string) (res ExecResult) {
	var username string
	err := mydb.DBConn().QueryRow(
		"select t.user_name from tbl_user_token t join tbl_user u on u.user_name=t.user_name "+
			"where t.token_hash=? and t.expire_at>NOW() and u.status=? limit 1",
		tokenHash, UserStatusNormal).Scan(&username)
	if err == sql.ErrNoRows {
		return notFoundResult("token无效或已过期")
	} else if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(username)
}

// DeleteToken : 注销token(退出登录)
func DeleteToken(tokenHash string) (res ExecResult) {
	if _, err := mydb.DBConn().Exec(
		"delete from tbl_user_token where token_hash=?", tokenHash); err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(nil)
}

// GetUserInfo : 查询用户信息
func GetUserInfo(username string) (res ExecResult) {
	user := TableUser{}
	err := mydb.DBConn().QueryRow(
		"select user_name,email,phone,"+
			"date_format(signup_at,'%Y-%m-%d %H:%i:%s'),date_format(last_active,'%Y-%m-%d %H:%i:%s'),status "+
			"from tbl_user where user_name=? limit 1", username).Scan(
		&user.Username, &user.Email, &user.Phone, &user.SignupAt, &user.LastActiveAt, &user.Status)
	if err == sql.ErrNoRows {
		return notFoundResult("用户不存在")
	} else if err != nil {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(user)
}

// UserExist : 查询用户是否存在
func UserExist(username string) (res ExecResult) {
	var one int
	err := mydb.DBConn().QueryRow(
		"select 1 from tbl_user where user_name=? limit 1", username).Scan(&one)
	if err != nil && err != sql.ErrNoRows {
		log.Println(err.Error())
		return errResult(err)
	}
	return okResult(map[string]bool{"exists": err == nil})
}
