package gcron

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func BenchmarkRedisZset(b *testing.B) {
	LoadConfig()
	redis := RedisInstance()
	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := `
			{
				"ExecTime":` + fmt.Sprint(rand.Int63n(time.Now().UnixNano())) + `,
				"Args":{
					"user_id":"12"
				},
				"Url":"http://127.0.0.1:1040",
				"Method":"post",
				"Header":{
					"Version":"1.0"
				},
				"TTL":100
			}`
		redis.Do("zadd", "test", rand.Uint64(), data)
	}
}
