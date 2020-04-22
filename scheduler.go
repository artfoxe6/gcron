package gcron

import (
	"fmt"
	"time"
)

type Scheduler struct {
	Ticker     *time.Ticker
	Stop       chan int
	JobManager *JobManager
}

//创建一个调度器
func NewScheduler() *Scheduler {
	s := &Scheduler{
		JobManager: NewJobManager(),
	}
	return s
}

//开始调度
func (s *Scheduler) Start() {

	s.Ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-s.Ticker.C:
				s.JobManager.HandleBuffer <- fmt.Sprint(time.Now().Unix())
			case <-s.Stop:
				s.Ticker.Stop()
			}
		}
	}()

	<-s.Stop
}

//分布式环境下，每秒只有一个调度器能够获取到调度机会
//func (s *Scheduler) AppendHandleBuffer() {
//	execTime := strconv.Itoa(int(time.Now().Unix()))
//	lockKey := "scheduler_" + execTime
//	ok := s.locker.Lock(lockKey)
//	if ok {
//		fmt.Println("获取锁成功")
//		s.JobManager.HandleBuffer <- execTime
//		s.locker.Unlock(lockKey)
//	} else {
//		fmt.Println("获取锁失败")
//	}
//}
