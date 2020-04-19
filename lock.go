package gcron

import (
	"github.com/gomodule/redigo/redis"
)

type Lock struct{}

func (l *Lock) Lock(key string) bool {
	ok, _ := redis.Bool(RedisInstance().Do("set", "lock"+key, 1, "EX 10", "NX"))
	return ok
}

func (l *Lock) Unlock(key string) {
	_, _ = redis.Bool(RedisInstance().Do("del", "lock"+key))
}
