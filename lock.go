package gcron

import "github.com/gomodule/redigo/redis"

type Lock struct{}

func (l *Lock) Lock(key string) bool {
	_, err := redis.String(RedisInstance().Do("set", "lock_"+key, 1, "EX", "5", "NX"))
	if err != nil {
		return false
	}
	return true
}

func (l *Lock) Unlock(key string) {
	_, _ = redis.Bool(RedisInstance().Do("del", "lock_"+key))
}
