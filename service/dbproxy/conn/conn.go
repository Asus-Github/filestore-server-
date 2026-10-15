package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"

	cfg "filestore-server/config"
)

var db *sql.DB

// InitDBConn : 初始化数据库连接池, 连接失败时直接返回错误
func InitDBConn() error {
	conn, err := sql.Open("mysql", dsn())
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(100)
	conn.SetMaxIdleConns(20)
	conn.SetConnMaxLifetime(30 * time.Minute)
	conn.SetConnMaxIdleTime(5 * time.Minute)

	// 数据库可能比服务晚启动(如docker-compose), 稍作重试
	for i := 0; i < 10; i++ {
		if err = conn.Ping(); err == nil {
			break
		}
		log.Printf("Failed to connect to mysql(%s), retrying: %v", cfg.MySQLHost, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to connect to mysql: %w", err)
	}
	db = conn
	return nil
}

// dsn : 在基础数据源上追加会话参数
// clientFoundRows: UPDATE 时返回匹配行数而非实际变更行数(新旧值相同也算成功)
// time_zone: 让 NOW()/CURRENT_TIMESTAMP 与服务所在时区一致, 避免容器内MySQL使用UTC导致时间错乱
func dsn() string {
	_, offset := time.Now().Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	tz := fmt.Sprintf("'%s%02d:%02d'", sign, offset/3600, (offset%3600)/60)
	return cfg.MySQLSource() + "&clientFoundRows=true&time_zone=" + url.QueryEscape(tz)
}

// DBConn : 返回数据库连接对象
func DBConn() *sql.DB {
	return db
}
