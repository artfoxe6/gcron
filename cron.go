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
func (cm *CronManager) Add(cronRule, taskJson string, locationOffset int) error {
	var err error
	taskData := TaskData{}
	err = json.Unmarshal([]byte(taskJson), &taskData)
	if err != nil {
		return errors.New("任务格式错误")
	}
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
		LocationOffset: locationOffset,
	}
	task.Minute, err = cronRuleParse(ruleItems[0], []int{0, 59})
	if err != nil {
		return errors.New("分钟" + err.Error())
	}
	task.Hour, err = cronRuleParse(ruleItems[1], []int{0, 23})
	if err != nil {
		return errors.New("小时" + err.Error())
	}
	task.Dom, err = cronRuleParse(ruleItems[2], []int{0, 23})
	if err != nil {
		return errors.New("日(月)" + err.Error())
	}
	task.Month, err = cronRuleParse(ruleItems[3], []int{0, 23})
	if err != nil {
		return errors.New("月" + err.Error())
	}
	task.Dow, err = cronRuleParse(ruleItems[4], []int{0, 23})
	if err != nil {
		return errors.New("周(月)" + err.Error())
	}
	fmt.Println(task.ComputeNextRunAt())
	//cm.Tasks = append(cm.Tasks, task)
	return nil
}

//任务列表
func (cm *CronManager) List() []*Task {
	return cm.Tasks
}

func (t *Task) ComputeNextRunAt() string {
	now := time.Now().In(time.FixedZone(t.LocationName, t.LocationOffset))
	nowCopy := now
	var (
		month  = int(now.Month())
		week   = int(now.Weekday())
		day    = now.Day()
		hour   = now.Hour()
		minute = now.Minute()
	)
	//当前月份不在规则中
	if !existsInArray(t.Month, month) {
		nextMonth := getMinValueInArray(t.Month, month)
		if nextMonth > month {
			nowCopy.AddDate(0, nextMonth-month, 0)
		} else {
			nowCopy.AddDate(0, nextMonth+12-month, 0)
		}
	}
	//当前周(月)不在规则中
	//todo  和日 or关系
	if !existsInArray(t.Dow, week) {
		nextWeek := getMinWeekInArray(t.Dow, week)
		nowCopy.AddDate(0, 0, nextWeek-week)
	}
	//当前日(月)不在规则中
	//todo  和周 or关系,, 还可能存在条件超越父级
	if !existsInArray(t.Dom, day) {
		nextDay := getMinValueInArray(t.Dom, day)
		if nextDay > day {
			nowCopy.AddDate(0, 0, nextDay-day)
		} else {
			currentMonthHasDays := getCurrentMonthHasDays(t.LocationName, t.LocationOffset)
			nowCopy.AddDate(0, 0, nextDay+(currentMonthHasDays)-day)
		}
	}
	//当前小时不在规则中
	if !existsInArray(t.Hour, hour) {
		nextHour := getMinValueInArray(t.Hour, hour)
		if nextHour > hour {
			nowCopy.Add(time.Duration(nextHour-hour) * time.Second * 3600)
		} else {
			nowCopy.Add(time.Duration(nextHour+24-hour) * time.Second * 3600)
		}
	}
	//当前分钟不在规则中
	if !existsInArray(t.Minute, minute) {
		nextMinute := getMinValueInArray(t.Minute, minute)
		if nextMinute > minute {
			nowCopy.Add(time.Duration(nextMinute-minute) * time.Second * 60)
		} else {
			nowCopy.Add(time.Duration(nextMinute+60-minute) * time.Second * 60)
		}
	}
	return nowCopy.Format("2006-01-02 15:04:05")
}
