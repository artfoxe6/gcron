package gcron

import (
	"encoding/json"
	"github.com/artfoxe6/cron_expression"
	"github.com/gomodule/redigo/redis"
	"log"
	"sync"
	"time"
)

//定时任务
type CronJob struct {
	CronExpr       string                 //定时任务cron表达式
	LastRunAt      int64                  //上次执行时间点 单位时间戳
	NextRunAt      int64                  //下次执行时间点 单位时间戳
	TTL            int64                  //任务能忍受的超时时间
	UUID           string                 //任务唯一标识
	Desc           string                 //任务描述
	LocationName   string                 //时区
	LocationOffset int                    //和UTC的偏差多少秒
	Args           map[string]interface{} //请求参数
	Url            string                 //请求地址
	Method         string                 //请求方法
	Header         map[string]string      //自定义header
}

//任务管理
type JobManager struct {
	WaitHandle chan int64 //由调度器派发的任务队列
	Locker     *Lock
	Processing []map[string]string //当前正在处理的任务列表
}

//创建一个任务管理器
func NewJobManager() *JobManager {
	return &JobManager{
		WaitHandle: make(chan int64, 1000),
		Locker:     &Lock{},
		Processing: make([]map[string]string, 0),
	}
}

//启动任务处理
func (jbm *JobManager) StartHandleJob() {
	for {
		select {
		case scheduleTime := <-jbm.WaitHandle:
			go func(scheduleTime int64) {
				//以当前时间戳为score在zset中筛选任务uuid
				uniqueIds, ok := jbm.FindJob(scheduleTime)
				if ok {
					for _, uniqueId := range uniqueIds {
						//通过任务uuid去hash中查找任务的具体数据
						job, yes := jbm.GetJobData(uniqueId)
						if !yes {
							continue
						}
						go jbm.Exec(job)
					}
				}
			}(scheduleTime)
		default:
			time.Sleep(time.Second)
		}
	}
}

//查找任务标识 任务标识存在一个zset中,执行时间作为分数
func (jbm *JobManager) FindJob(jobTime int64) ([]string, bool) {
	uniqueIds, err := redis.Strings(RedisInstance().Do("ZRANGEBYSCORE", "test", 0, jobTime))
	if err != nil {
		return nil, false
	}
	if len(uniqueIds) == 0 {
		return nil, false
	}
	return uniqueIds, true
}

//获取任务的具体数据  具体任务存在一个hash中
func (jbm *JobManager) GetJobData(uniqueId string) (*CronJob, bool) {
	jobString, err := redis.String(RedisInstance().Do("HGET", "test_hash", uniqueId))
	if err != nil {
		return nil, false
	}
	job := &CronJob{}
	err = json.Unmarshal([]byte(jobString), job)
	if err != nil {
		return nil, false
	}
	return job, true
}

var mutex sync.Mutex

//执行任务
func (jbm *JobManager) Exec(job *CronJob) {
	if (job.NextRunAt + job.TTL) < time.Now().Unix() {
		log.Println("任务超时，取消执行,并需记录日志")
		return
	}
	if success := jbm.Locker.Lock(job.UUID); !success {
		//获取锁失败,跳过,说明有其他线程正在处理该任务
		return
	}
	defer jbm.Locker.Unlock(job.UUID)
	jobByteData, err := json.Marshal(*job)
	if err != nil {
		ErrLog("任务的格式错误:" + job.UUID)
		return
	}
	h := httpData(jobByteData)
	body, code, _ := h.SendHttp()
	RunLog(code, string(body))
	//处理结束后重新计算任务的下次执行时间
	expr := cron_expression.NewExpression(job.CronExpr, job.LocationName, job.LocationOffset)
	dst := make([]string, 0)
	err = expr.Next(time.Now(), 1, &dst)
	if err != nil {
		ErrLog("任务的下次执行时间计算出错" + job.UUID)
		return
	}
	job.LastRunAt = job.NextRunAt
	t, err := time.Parse("2006-01-02 15:04:05", dst[0])
	if err != nil {
		ErrLog("下次执行时间解析出错" + job.UUID)
		return
	}
	job.NextRunAt = t.Unix()
	_, err = RedisInstance().Do("ZADD", "test", job.NextRunAt, job.UUID)
	if err != nil {
		ErrLog("ZADD error" + job.UUID)
		return
	}
	jobByte, err := json.Marshal(job)
	if err != nil {
		ErrLog("任务更新错误" + job.UUID)
		return
	}
	_, err = RedisInstance().Do("HSET", "test_hash", job.UUID, string(jobByte))
	if err != nil {
		ErrLog("HSET error" + job.UUID)
		return
	}
}
