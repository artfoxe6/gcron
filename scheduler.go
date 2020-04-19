package gcron

import (
	"strconv"
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
				s.JobManager.HandleBuffer <- strconv.Itoa(int(time.Now().Unix()))
			case <-s.Stop:
				s.Ticker.Stop()
			}
		}
	}()

	<-s.Stop
}
