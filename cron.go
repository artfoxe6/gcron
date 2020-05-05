package gcron

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

//linux下crontab的golang实现

type Task struct {
	Minute         []int      //分 规则
	Hour           []int      //时 规则
	Dom            []int      //日 规则
	Month          []int      //月 规则
	Dow            []int      //周 规则
	JoinAt         time.Time  //任务添加时间
	LastRunAt      *time.Time //上次执行时间
	NextRunAt      *time.Time //执行时间
	ExecTimes      int        //执行次数
	TaskData       TaskData   //任务数据
	LocationOffset int        //当前时区与UTC的偏移秒数
	LocationName   string     //当前时区名
}

//具体任务内容
type TaskData struct {
	Args            map[string]interface{} //请求参数
	Url             string                 //请求地址
	Method          string                 //请求方法
	Header          map[string]string      //自定义header
	AllowMaxTimeout int64                  //任务能忍受的超时时间
	UUID            string                 //任务唯一标识
	Desc            string                 //任务描述
}

//任务管理
type CronManager struct {
	Tasks []*Task
}

//创建任务管理器
func NewCronManager() *CronManager {
	return &CronManager{
		Tasks: make([]*Task, 100),
	}
}

//添加任务
func (cm *CronManager) Add(cronRule, taskJson string) error {
	var err error
	taskData := TaskData{}
	err = json.Unmarshal([]byte(taskJson), &taskData)
	//if err != nil {
	//	return errors.New("任务格式错误")
	//}
	ruleItems := strings.Split(cronRule, " ")
	if len(ruleItems) != 5 {
		return errors.New("cron规则格式错误")
	}
	task := &Task{
		JoinAt:         time.Now(),
		LastRunAt:      nil,
		NextRunAt:      nil,
		ExecTimes:      0,
		TaskData:       taskData,
		LocationOffset: 8 * 3600,
		LocationName:   "CST",
	}
	task.Minute, err = cronRuleParse(ruleItems[0], []int{0, 59})
	if err != nil {
		return errors.New("分钟" + err.Error())
	}
	task.Hour, err = cronRuleParse(ruleItems[1], []int{0, 23})
	if err != nil {
		return errors.New("小时" + err.Error())
	}
	task.Dom, err = cronRuleParse(ruleItems[2], []int{1, 31})
	if err != nil {
		return errors.New("日(月)" + err.Error())
	}
	task.Month, err = cronRuleParse(ruleItems[3], []int{1, 12})
	if err != nil {
		return errors.New("月" + err.Error())
	}
	task.Dow, err = cronRuleParse(ruleItems[4], []int{0, 6})
	if err != nil {
		return errors.New("周(月)" + err.Error())
	}
	fmt.Println(task.ComputeNextRunAt(time.Now()))
	//cm.Tasks = append(cm.Tasks, task)
	return nil
}

//任务列表
func (cm *CronManager) List() []*Task {
	return cm.Tasks
}

func (t *Task) ComputeNextRunAt(current time.Time) string {
	now := current.In(time.FixedZone(t.LocationName, t.LocationOffset))
	next := struct {
		year   int
		month  int
		day    int
		hour   int
		minute int
		week   int
	}{
		year:  now.Year(),
		month: int(now.Month()),
		day:   now.Day(),
		hour:  now.Hour(),
		week:  int(now.Weekday()),
	}
	change := struct {
		month  int
		day    int
		hour   int
		minute int
	}{}
	var (
		month  = int(now.Month())
		week   = int(now.Weekday())
		day    = now.Day()
		hour   = now.Hour()
		minute = now.Minute()
	)
	//next month
	if !existsInArray(t.Month, month) {
		next.month = getMinValueInArray(t.Month, month)
		change.month = 1
	}
	//todo  和日 or关系
	//next week
	if !existsInArray(t.Dow, week) {
		//nextWeek := getMinWeekInArray(t.Dow, week)
		//nowCopy =  nowCopy.AddDate(0, 0, nextWeek-week)
	}
	//todo  和周 or关系,
	// next day
	if change.month == 1 {
		next.day = t.Dom[0]
		change.day = 1
	} else {
		if !existsInArray(t.Dom, day) {
			next.day = getMinValueInArray(t.Dom, day)
			change.day = 1
		}
	}
	//next hour
	if change.day == 1 {
		next.hour = t.Hour[0]
		change.hour = 1
	} else {
		if !existsInArray(t.Hour, hour) {
			next.hour = getMinValueInArray(t.Hour, hour)
			change.hour = 1
		}
	}
	//next minute
	if change.hour == 1 {
		next.minute = t.Minute[0]
	} else {
		if !existsInArray(t.Minute, minute) {
			next.minute = getMinValueInArray(t.Minute, minute)
		}
	}

	res := time.Date(next.year, time.Month(next.month), next.day, next.hour, next.minute, 0, 0, now.Location())
	currentReally := time.Now().In(time.FixedZone(t.LocationName, t.LocationOffset))
	if res.Unix() < currentReally.Unix() {
		//需要条件回归
		fmt.Println("error")
	}
	fmt.Println("周", t.Dow)
	fmt.Println("月", t.Month)
	fmt.Println("日", t.Dom)
	fmt.Println("时", t.Hour)
	fmt.Println("分", t.Minute)
	return res.Format("2006-01-02 15:04:05")
}
