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
	ExecTime int64                  //执行时间
	Args     map[string]interface{} //请求参数
	Url      string                 //请求地质
	Method   string                 //请求方法
	Header   map[string]string      //自定义header
	TTL      int64                  //任务能忍受的超时时间
}

type JobManager struct {
	InBuffer     chan []byte
	HandleBuffer chan string
}

var jobManagerInstance *JobManager

//创建一个任务管理器
func NewJobManager() *JobManager {
	if jobManagerInstance != nil {
		return jobManagerInstance
	}
	jobManagerInstance = &JobManager{
		InBuffer:     make(chan []byte, 1000),
		HandleBuffer: make(chan string, 1000),
	}
	go func() {
		for {
			select {
			case jobData := <-jobManagerInstance.InBuffer:
				go jobManagerInstance.insertJob(jobData)
			case execTime := <-jobManagerInstance.HandleBuffer:
				go jobManagerInstance.handle(execTime)
			default:
				time.Sleep(time.Second * 2)
			}
		}
	}()
	return jobManagerInstance
}

//把任务插入到redis中
func (tm *JobManager) insertJob(jobdata []byte) {
	job := Job{}
	_ = json.Unmarshal(jobdata, &job)
	h := md5.New()
	_, _ = io.WriteString(h, string(jobdata))
	RedisInstance().Do("sadd", job.ExecTime, h.Sum(nil), jobdata)
}

//重试
func (tm *JobManager) retry(execTime string) {
	time.Sleep(time.Second * 10)
	tm.HandleBuffer <- execTime
}

//任务进入通道
func (tm *JobManager) Add(jsonData string) {
	job := Job{}
	err := json.Unmarshal([]byte(jsonData), &job)
	if err != nil {
		fmt.Println("参数格式不符合 map[string]interface{}")
	}
	tm.InBuffer <- []byte(jsonData)
}

//执行任务
//execTime: time[_retry_count]
func (tm *JobManager) handle(execTime string) {
	num, err := redis.Int64(RedisInstance().Do("scard", execTime))
	if err != nil {
		log.Println("发送到sentry")
		return
	}
	defer func() {
		if err := recover(); err == nil {
			RedisInstance().Do("del", execTime)
		}
	}()
	//如果集合中超过1000个任务，采用分段获取
	if num < 1000 {
		values, err := redis.Values(RedisInstance().Do("smembers", execTime))
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
		values, err := redis.Values(RedisInstance().Do("srandmember", execTime, 1000))
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
	if (job.TTL + job.ExecTime) < time.Now().Unix() {
		log.Println("任务超时，发送到sentry")
		return
	}
	h := httpData(jobdata)
	h.SendHttp()
}
