package config

import "fmt"

var (
	// MySQLUser : 数据库用户名
	MySQLUser = envString("FS_MYSQL_USER", "test")
	// MySQLPassword : 数据库密码
	MySQLPassword = envString("FS_MYSQL_PASSWORD", "test")
	// MySQLHost : 数据库地址(ip:port), 可被 --dbhost 参数覆盖
	MySQLHost = envString("FS_MYSQL_HOST", "127.0.0.1:3306")
	// MySQLDB : 数据库名
	MySQLDB = envString("FS_MYSQL_DB", "fileserver")
	// RedisHost : redis地址(ip:port), 可被 --cachehost 参数覆盖
	RedisHost = envString("FS_REDIS_HOST", "127.0.0.1:6379")
	// RedisPassword : redis密码
	RedisPassword = envString("FS_REDIS_PASSWORD", "testupload")
)

// MySQLSource : 要连接的数据库源
// utf8mb4 支持emoji等4字节字符; parseTime=false 让时间以字符串形式返回, loc=Local 保证时区一致
func MySQLSource() string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=false&loc=Local",
		MySQLUser, MySQLPassword, MySQLHost, MySQLDB)
}
