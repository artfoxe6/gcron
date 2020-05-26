package gcron

import (
	"encoding/json"
	"github.com/artfoxe6/cron_expression"
	"github.com/gomodule/redigo/redis"
	"log"
	"strconv"
	"time"
)

//定时任务
type CronJob struct {
	CronExpr       string                 //定时任务cron表达式
	LastRunAt      int64                  //上次执行时间点 单位时间戳
	NextRunAt      int64                  //下次执行时间点 单位时间戳
	TTL            int64                  //任务能忍受的超时时间
	Id             string                 //任务唯一标识
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
	JobHandling chan int64 //等待执行的任务
	Locker      *Lock
	Stop        chan bool
}

//创建一个任务管理器
func NewJobManager() *JobManager {
	return &JobManager{
		JobHandling: make(chan int64, 100),
		Locker:      &Lock{},
		Stop:        make(chan bool, 0),
	}
}

//启动任务处理
//当节点晋升为leader时,将关闭拉取任务,已经拉取的任务会继续执行完
func (jbm *JobManager) Start() {
	//一个协程负责拉取任务
	go jbm.PullJob()
	//一个协程负责处理任务
	go jbm.DoJob()
}

func (jbm *JobManager) DoJob() {
	for {
		select {
		case jobId := <-jbm.JobHandling:
			//通过任务id去hash中查找任务的具体数据
			job, ok := jbm.GetJobData(jobId)
			if !ok {
				continue
			}
			go jbm.Exec(job)
		default:
			time.Sleep(time.Second)
		}
	}
}

//从redis中获取一个任务
//遇到错误或者暂时没有任务,就休息1秒钟,有任务的情况下持续拉取直到没有任务或者 JobHandling 装满
func (jbm *JobManager) PullJob() {
	for {
		select {
		case <-jbm.Stop:
			return
		default:
			jobId, err := redis.String(RedisInstance().Do("SPOP", RedisConfig.Ready))
			if err != nil || jobId == "" {
				time.Sleep(time.Second)
				continue
			}
			id, err := strconv.ParseInt(jobId, 10, 64)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			jbm.JobHandling <- id
		}
	}

}

//获取任务的具体数据  具体任务存在一个hash中
func (jbm *JobManager) GetJobData(jobId int64) (*CronJob, bool) {
	jobString, err := redis.String(RedisInstance().Do("HGET", "test_hash", jobId))
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

//计算任务的下一个执行时间点
func (job *CronJob) CalculatorNextAt() (string, error) {
	//处理结束后重新计算任务的下次执行时间
	expr := cron_expression.NewExpression(job.CronExpr, job.LocationName, job.LocationOffset)
	dst := make([]string, 0)
	err := expr.Next(time.Now(), 1, &dst)
	if err != nil {
		ErrLog("任务的下次执行时间计算出错" + job.Id)
		return "", err
	}
	return dst[0], nil
}

//执行任务
func (jbm *JobManager) Exec(job *CronJob) {
	if (job.NextRunAt + job.TTL) < time.Now().Unix() {
		log.Println("任务超时，取消执行,并需记录日志")
		return
	}
	if success := jbm.Locker.Lock(job.Id); !success {
		//获取锁失败,跳过,说明有其他线程正在处理该任务
		return
	}
	defer jbm.Locker.Unlock(job.Id)
	jobByteData, err := json.Marshal(*job)
	if err != nil {
		ErrLog("任务的格式错误:" + job.Id)
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
		ErrLog("任务的下次执行时间计算出错" + job.Id)
		return
	}
	job.LastRunAt = job.NextRunAt
	t, err := time.Parse("2006-01-02 15:04:05", dst[0])
	if err != nil {
		ErrLog("下次执行时间解析出错" + job.Id)
		return
	}
	job.NextRunAt = t.Unix()
	_, err = RedisInstance().Do("ZADD", "test", job.NextRunAt, job.Id)
	if err != nil {
		ErrLog("ZADD error" + job.Id)
		return
	}
	jobByte, err := json.Marshal(job)
	if err != nil {
		ErrLog("任务更新错误" + job.Id)
		return
	}
	_, err = RedisInstance().Do("HSET", "test_hash", job.Id, string(jobByte))
	if err != nil {
		ErrLog("HSET error" + job.Id)
		return
	}
}
