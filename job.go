package gcron

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"io"
	"log"
	"time"
)

type Job struct {
	Type           int                    //执行类型 1 定时任务  2 延时任务
	CronExpr       string                 //cron 表达式
	At             int64                  //执行时间单位秒
	Args           map[string]interface{} //请求参数
	Url            string                 //请求地址
	Method         string                 //请求方法
	Header         map[string]string      //自定义header
	TTL            int64                  //任务能忍受的超时时间
	UUID           string                 //任务唯一标识
	Desc           string                 //任务描述
	LocationName   string                 //时区
	LocationOffset int                    //和UTC的偏差多少秒
}

type JobManager struct {
	InBuffer     chan []byte
	HandleBuffer chan string
	Locker       *Lock
}

var jobManager *JobManager

//创建一个任务管理器
func NewJobManager() *JobManager {
	if jobManager != nil {
		return jobManager
	}
	jobManager = &JobManager{
		InBuffer:     make(chan []byte, 1000),
		HandleBuffer: make(chan string, 1000),
		Locker:       &Lock{},
	}
	go func() {
		for {
			select {
			case jobData := <-jobManager.InBuffer:
				go jobManager.insertJob(jobData)
			case currentTime := <-jobManager.HandleBuffer:
				go jobManager.scheduling(currentTime)
			default:
				time.Sleep(time.Second * 2)
			}
		}
	}()
	return jobManager
}

//把任务插入到redis中
func (jm *JobManager) insertJob(jobdata []byte) {
	job := Job{}
	_ = json.Unmarshal(jobdata, &job)
	h := md5.New()
	_, _ = io.WriteString(h, string(jobdata))
	RedisInstance().Do("sadd", job.At, h.Sum(nil), jobdata)
}

//重试
func (jm *JobManager) retry(execTime string) {
	time.Sleep(time.Second * 10)
	jm.HandleBuffer <- execTime
}

//任务进入通道
func (jm *JobManager) Add(jsonData string) {
	job := Job{}
	err := json.Unmarshal([]byte(jsonData), &job)
	if err != nil {
		fmt.Println("参数格式不符合 map[string]interface{}")
	}
	jm.InBuffer <- []byte(jsonData)
}

//执行任务
//execTime: time[_retry_count]
func (jm *JobManager) scheduling(currentTime string) {
	lockKey := "scheduling_" + currentTime
	ok := jm.Locker.Lock(lockKey)
	//分布式环境下，只有获取到锁的节点能够处理任务
	if !ok {
		return
	}
	num, err := redis.Int64(RedisInstance().Do("scard", currentTime))
	if err != nil {
		log.Println("发送到sentry")
		return
	}
	defer func() {
		if err := recover(); err == nil {
			RedisInstance().Do("del", currentTime)
		}
	}()
	//如果集合中超过1000个任务，采用分段获取
	if num < 1000 {
		values, err := redis.Values(RedisInstance().Do("smembers", currentTime))
		if err != nil {
			log.Println("发送到sentry")
			return
		}
		for _, job := range values {
			go HandleJob(job.([]byte))
		}
	}
	for {
		num -= 1000
		values, err := redis.Values(RedisInstance().Do("srandmember", currentTime, 1000))
		if err != nil {
			log.Println("发送到sentry")
			return
		}
		for _, job := range values {
			go HandleJob(job.([]byte))
		}
		if num <= 0 {
			break
		}
	}

}

//执行任务
func HandleJob(jobdata []byte) {
	job := Job{}

	err := json.Unmarshal(jobdata, &job)
	if err != nil {
		log.Println("数据格式错误")
		return
	}
	if (job.TTL + job.At) < time.Now().Unix() {
		log.Println("任务超时，发送到sentry")
		return
	}
	h := httpData(jobdata)
	h.SendHttp()
}
