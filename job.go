package gcron

import (
	"encoding/json"
	"fmt"
	"github.com/artfoxe6/cron_expression"
	"github.com/gomodule/redigo/redis"
	"log"
	"strconv"
	"strings"
	"time"
)

//定时任务
type CronJob struct {
	CronExpr string //定时任务cron表达式
	//LastRunAt      int64                  //上次执行时间点 单位时间戳
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
	JobHandling chan string //等待执行的任务
	Locker      *Lock
	Pulling     bool
}

//创建一个任务管理器
func NewJobManager() *JobManager {
	jbm := &JobManager{
		JobHandling: make(chan string, 100),
		Locker:      &Lock{},
	}
	//负责处理任务的协程
	go jbm.DoJob()
	return jbm
}

//启动任务处理
func (jbm *JobManager) Start() {
	//保证当前只能有一个拉取任务的协程
	if jbm.Pulling == true {
		return
	}
	//负责拉取任务协程
	go jbm.PullJob()
}

//暂定任务处理
func (jbm *JobManager) Stop() {
	jbm.Pulling = false
}

//处理拉取的任务
func (jbm *JobManager) DoJob() {
	fmt.Println("开启处理任务")
	for {
		select {
		case jobAndUnix := <-jbm.JobHandling:
			//通过任务id去hash中查找任务的具体数据
			job, ok := jbm.GetJobData(jobAndUnix)
			if !ok {
				continue
			}
			go jbm.Exec(job)
		default:
			time.Sleep(time.Second)
		}
	}
}

//负责拉取任务协程
//从redis中获取一个任务
//遇到错误或者暂时没有任务,就休息1秒钟,有任务的情况下持续拉取直到没有任务或者 JobHandling 装满
func (jbm *JobManager) PullJob() {
	fmt.Println("开始任务拉取")
	jbm.Pulling = true
	for {
		if jbm.Pulling == false {
			fmt.Println("停止任务拉取")
			return
		}
		select {
		default:
			jobAndUnix, err := redis.String(RedisInstance().Do("SPOP", RedisConfig.Ready))
			if err != nil || jobAndUnix == "" {
				time.Sleep(time.Second)
				continue
			}
			jbm.JobHandling <- jobAndUnix
		}
	}

}

//获取任务的具体数据  具体任务存在一个hash中
func (jbm *JobManager) GetJobData(jobAndUnix string) (*CronJob, bool) {
	temp := strings.Split(jobAndUnix, "_")
	id, err := strconv.ParseInt(temp[0], 10, 64)
	if err != nil {
		return nil, false
	}
	jobString, err := redis.String(RedisInstance().Do("HGET", "test_hash", id))
	if err != nil {
		return nil, false
	}
	job := &CronJob{}
	err = json.Unmarshal([]byte(jobString), job)
	if err != nil {
		return nil, false
	}
	//下一个执行时间点
	at, err := strconv.ParseInt(temp[1], 10, 64)
	if err != nil {
		return nil, false
	}
	job.NextRunAt = at
	return job, true
}

//执行任务
func (jbm *JobManager) Exec(job *CronJob) {
	if (job.NextRunAt + job.TTL) < time.Now().Unix() {
		log.Println("任务超时，取消执行,并需记录日志")
		return
	}
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
	nextAt, err := expr.Next(time.Now())
	if err != nil {
		ErrLog("任务的下次执行时间计算出错" + job.Id)
		return
	}
	_, err = RedisInstance().Do("ZADD", RedisConfig.JobMeta, nextAt.Unix(), job.Id)
	if err != nil {
		ErrLog("ZADD error" + job.Id)
		return
	}
}
