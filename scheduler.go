package gcron

import (
	"strconv"
	"time"
)

type Scheduler struct {
	Ticker     *time.Ticker
	Stop       chan int
	JobManager *JobManager
	locker     *Lock
}

//创建一个调度器
func NewScheduler() *Scheduler {
	s := &Scheduler{
		JobManager: NewJobManager(),
		locker:     &Lock{},
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
				s.AppendHandleBuffer()
			case <-s.Stop:
				s.Ticker.Stop()
			}
		}
	}()

	<-s.Stop
}

//分布式环境下，每秒只有一个调度器能够获取到调度机会
func (s *Scheduler) AppendHandleBuffer() {
	execTime := strconv.Itoa(int(time.Now().Unix()))
	lockKey := "scheduler_" + execTime
	ok := s.locker.Lock(lockKey)
	if ok {
		s.JobManager.HandleBuffer <- execTime
		s.locker.Unlock(lockKey)
	}
}
