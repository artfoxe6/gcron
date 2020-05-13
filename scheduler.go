package gcron

import (
	"time"
)

//调度器
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
	s.JobManager.StartHandleJob()
	s.Ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-s.Ticker.C:
				//每一分钟触发一次任务调度
				if time.Now().Second() == 0 {
					s.JobManager.WaitHandle <- time.Now().Unix()
				}
			case <-s.Stop:
				s.Ticker.Stop()
			}
		}
	}()

	<-s.Stop
}
