package gcron

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	LoadConfig()
	p := make(chan int, 10)
	for i := 0; i < 10; i++ {

		go func(i int) {
			locker := &Lock{}
			ok := locker.Lock("key")
			if !ok {
				//fmt.Println("获取🔓锁失败" + strconv.Itoa(i))
			} else {
				time.Sleep(time.Second)
				fmt.Println("get lock " + strconv.Itoa(i))
				locker.Unlock("key")
			}
			p <- i
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-p
	}
}
