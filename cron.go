package gcron

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

//标准的crontab的golang实现,不支持 @everyday...等扩展语法
//如果dom日(月) 和 dow周(月)都不为*, 那么任何一方出现 / 每隔语法,两者关系变为且,否则为或
//参考 https://crontab.guru/
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
	CronRule       string     //原始的cron规则
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

//规则键位
const (
	MinuteKey = iota
	HourKey
	DomKey
	MonthKey
	DowKey
)

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
		CronRule:       cronRule,
	}
	task.Minute, err = cronRuleParse(ruleItems[MinuteKey], []int{0, 59})
	if err != nil {
		return errors.New("分钟" + err.Error())
	}
	task.Hour, err = cronRuleParse(ruleItems[HourKey], []int{0, 23})
	if err != nil {
		return errors.New("小时" + err.Error())
	}
	task.Dom, err = cronRuleParse(ruleItems[DomKey], []int{1, 31})
	if err != nil {
		return errors.New("日(月)" + err.Error())
	}
	task.Month, err = cronRuleParse(ruleItems[MonthKey], []int{1, 12})
	if err != nil {
		return errors.New("月" + err.Error())
	}
	task.Dow, err = cronRuleParse(ruleItems[DowKey], []int{0, 6})
	if err != nil {
		return errors.New("周(月)" + err.Error())
	}
	fmt.Println(task.ComputeNextRunAt(time.Now()))
	//cm.Tasks = append(cm.Tasks, task)
	return nil
}

//时间点变更记录
type change struct {
	month  int
	day    int
	hour   int
	minute int
}

//下一个执行时间点
type nextAt struct {
	year   int
	month  int
	day    int
	hour   int
	minute int
	week   int
}

func (t *Task) nextMonth(now time.Time, change *change, nextAt *nextAt, nextNum int) {
	current := int(now.Month())
	if !existsInArray(t.Month, current) {
		change.month = 1
		nextAt.month = getMinValueInArray(t.Month, current)
	} else {
		if nextNum == 0 {
			nextAt.month = current
		} else {
			change.month = 1
			nextAt.month = getMinValueInArray(t.Month, current)
		}
	}
	t.nextWeek(now, change, nextAt)
}
func (t *Task) nextDay(now time.Time, change *change, nextAt *nextAt, nextNum int) {
	current := now.Day()
	if change.month == 1 {
		nextAt.day = t.Dom[0]
		change.day = 1
	} else {
		if !existsInArray(t.Dom, current) {
			nextAt.day = getMinValueInArray(t.Dom, current)
			change.day = 1
		} else {
			if nextNum == 0 {
				nextAt.day = current
			} else {
				nextAt.day = getMinValueInArray(t.Dom, current)
				change.day = 1
			}
		}
		if nextAt.day < current {
			t.nextMonth(now, change, nextAt, 1)
		}
	}
}
func (t *Task) nextHour(now time.Time, change *change, nextAt *nextAt, nextNum int) {
	current := now.Hour()
	if change.day == 1 {
		nextAt.hour = t.Hour[0]
		change.hour = 1
	} else {
		if !existsInArray(t.Hour, current) {
			nextAt.hour = getMinValueInArray(t.Hour, current)
			change.hour = 1
		} else {
			if nextNum == 0 {
				nextAt.hour = current
			} else {
				nextAt.hour = getMinValueInArray(t.Hour, current)
				change.hour = 1
			}
		}
		if nextAt.hour < current {
			t.nextDay(now, change, nextAt, 1)
		}
	}
}
func (t *Task) nextMinute(now time.Time, change *change, nextAt *nextAt, nextNum int) {
	current := now.Minute()
	if change.hour == 1 {
		nextAt.minute = t.Minute[0]
	} else {
		nextAt.minute = getMinValueInArray(t.Minute, current)
		if nextAt.minute < current {
			t.nextHour(now, change, nextAt, 1)
		}
	}
}

//todo: dom 交集并集
func (t *Task) nextWeek(now time.Time, change *change, nextAt *nextAt) {
	ruleItems := strings.Split(t.CronRule, " ")
	if ruleItems[DowKey] == "*" {
		return
	}
	//dom和dow任意一个存在 间隔符 / 将形成交集
	days := getDayByWeek(now.Year(), nextAt.month, t.Dow, t.LocationName, t.LocationOffset)
	if strings.ContainsRune(ruleItems[DowKey], '/') || strings.ContainsRune(ruleItems[DomKey], '/') {
		t.Dom = arrayIntersect(t.Dom, days)
	} else {
		t.Dom = arrayMerge(t.Dom, days)
	}
}

func (t *Task) ComputeNextRunAt(current time.Time) string {
	now := current.In(time.FixedZone(t.LocationName, t.LocationOffset))

	var (
		change = &change{}
		nextAt = &nextAt{}
	)

	t.nextMonth(now, change, nextAt, 0)
	t.nextDay(now, change, nextAt, 0)
	t.nextHour(now, change, nextAt, 0)
	t.nextMinute(now, change, nextAt, 0)

	res := time.Date(now.Year(), time.Month(nextAt.month), nextAt.day, nextAt.hour, nextAt.minute, 0, 0, now.Location())
	fmt.Println("周", t.Dow)
	fmt.Println("月", t.Month)
	fmt.Println("日", t.Dom)
	fmt.Println("时", t.Hour)
	fmt.Println("分", t.Minute)
	return res.Format("2006-01-02 15:04:05")
}
