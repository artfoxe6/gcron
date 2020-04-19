package gcron

import (
	"gcron/config"
	"github.com/gomodule/redigo/redis"
	"log"
	"time"
)

var isLoad = false
var redisPool *redis.Pool

func createRedisPool() {
	cache := config.Redis
	redisPool = &redis.Pool{
		Dial: func() (con redis.Conn, err error) {
			con, err = redis.Dial("tcp", cache.Host,
				redis.DialPassword(cache.Password),
				redis.DialDatabase(cache.Db),
				redis.DialConnectTimeout(time.Second*time.Duration(cache.Timeout)),
				redis.DialReadTimeout(time.Second*time.Duration(cache.Timeout)),
				redis.DialWriteTimeout(time.Second*time.Duration(cache.Timeout)))
			if err != nil {
				log.Fatalln("Redis连接错误", err.Error())
			}
			return con, err
		},
		MaxIdle:         cache.MaxIdle,
		MaxActive:       cache.MaxActive,
		IdleTimeout:     cache.IdleTimeout,
		Wait:            true,
		MaxConnLifetime: 0,
	}
	isLoad = true
}

func RedisInstance() redis.Conn {
	if !isLoad {
		createRedisPool()
	}
	return redisPool.Get()
}
