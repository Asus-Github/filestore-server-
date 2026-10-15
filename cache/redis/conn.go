package redis

import (
	"log"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"

	cfg "filestore-server/config"
)

var (
	pool     *redis.Pool
	poolOnce sync.Once
)

// newRedisPool : 创建redis连接池
func newRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     30,
		MaxActive:   50,
		IdleTimeout: 300 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			// 1. 打开连接
			c, err := redis.Dial("tcp", cfg.RedisHost,
				redis.DialConnectTimeout(3*time.Second),
				redis.DialReadTimeout(5*time.Second),
				redis.DialWriteTimeout(5*time.Second),
			)
			if err != nil {
				log.Println("redis dial failed:", err)
				return nil, err
			}

			// 2. 访问认证(未配置密码时跳过)
			if cfg.RedisPassword != "" {
				if _, err = c.Do("AUTH", cfg.RedisPassword); err != nil {
					log.Println("redis auth failed:", err)
					c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := conn.Do("PING")
			return err
		},
	}
}

// RedisPool : 获取redis连接池(首次使用时按当前配置创建, 保证命令行参数已生效)
func RedisPool() *redis.Pool {
	poolOnce.Do(func() {
		pool = newRedisPool()
	})
	return pool
}

// Ping : 检查redis是否可用
func Ping() error {
	conn := RedisPool().Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	return err
}
