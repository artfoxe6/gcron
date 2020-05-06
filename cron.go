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
	DomCopy        []int      //日 规则拷贝
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
	task.DomCopy = make([]int, len(task.Dom))
	_ = copy(task.DomCopy, task.Dom)
	task.Month, err = cronRuleParse(ruleItems[MonthKey], []int{1, 12})
	if err != nil {
		return errors.New("月" + err.Error())
	}
	task.Dow, err = cronRuleParse(ruleItems[DowKey], []int{0, 6})
	if err != nil {
		return errors.New("周(月)" + err.Error())
	}
	task.ComputeNextRunAt(time.Now(), 5)
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

func (t *Task) nextMonth(now time.Time, change *change, nextAt *nextAt, jump *string) bool {
	change.month = 0
	change.day = 0
	change.hour = 0
	change.minute = 0
	nextAt.year = now.Year()
	current := int(now.Month())
	if *jump == "month" {
		change.month = 1
		nextAt.month = getMinValueInArray(t.Month, current)
		if nextAt.month <= current {
			nextAt.year += 1
		}
	} else {
		nextAt.month = current
		if !existsInArray(t.Month, current) {
			nextAt.month = getMinValueInArray(t.Month, current)
			change.month = 1
		}
	}
	t.Dom = make([]int, len(t.DomCopy))
	_ = copy(t.Dom, t.DomCopy)
	t.weekToDay(now, change, nextAt)
	return true
}
func (t *Task) nextDay(now time.Time, change *change, nextAt *nextAt, jump *string) bool {
	current := now.Day()
	if *jump != "day" {
		if change.month == 1 {
			nextAt.day = t.Dom[0]
			if nextAt.day != current {
				change.day = 1
			}
			return true
		}
		if !existsInArray(t.Dom, current) {
			nextAt.day = getMinValueInArray(t.Dom, current)
			if nextAt.day < current {
				*jump = "month"
				return false
			}
			if nextAt.day != current {
				change.day = 1
			}
		} else {
			nextAt.day = current
		}
	} else {
		nextAt.day = getMinValueInArray(t.Dom, current)
		if nextAt.day <= current {
			*jump = "month"
			return false
		}
		change.day = 1
	}
	return true
}
func (t *Task) nextHour(now time.Time, change *change, nextAt *nextAt, jump *string) bool {
	current := now.Hour()
	if *jump != "hour" {
		if change.day == 1 || change.month == 1 {
			nextAt.hour = t.Hour[0]
			if nextAt.hour != current {
				change.hour = 1
			}
			return true
		}
		if !existsInArray(t.Hour, current) {
			nextAt.hour = getMinValueInArray(t.Hour, current)
			if nextAt.hour < current {
				*jump = "day"
				return false
			}
			if nextAt.hour != current {
				change.hour = 1
			}
		} else {
			nextAt.hour = current
		}
	} else {
		nextAt.hour = getMinValueInArray(t.Hour, current)
		if nextAt.hour <= current {
			*jump = "day"
			return false
		}
		change.hour = 1
	}
	return true
}
func (t *Task) nextMinute(now time.Time, change *change, nextAt *nextAt, jump *string) bool {
	current := now.Minute()
	if change.hour == 1 || change.day == 1 || change.month == 1 {
		nextAt.minute = t.Minute[0]
	} else {
		nextAt.minute = getMinValueInArray(t.Minute, current)
		if nextAt.minute <= current {
			*jump = "hour"
			return false
		}
	}
	return true
}

//dow转dom
func (t *Task) weekToDay(now time.Time, change *change, nextAt *nextAt) {
	ruleItems := strings.Split(t.CronRule, " ")
	if ruleItems[DowKey] == "*" {
		return
	}
	//dom和dow任意一个存在 间隔符 / 将形成交集
	days := getDayByWeek(now.Year(), nextAt.month, t.Dow, t.LocationName, t.LocationOffset)
	if strings.ContainsRune(ruleItems[DowKey], '/') || strings.ContainsRune(ruleItems[DomKey], '/') {
		t.Dom = arrayIntersect(t.Dom, days)
	} else {
		if ruleItems[DomKey] != "*" {
			t.Dom = arrayMerge(t.Dom, days)
		} else {
			t.Dom = days
		}

	}
}

//计算下一个执行日期
func (t *Task) ComputeNextRunAt(current time.Time, nextNum int) string {
	now := current.In(time.FixedZone(t.LocationName, t.LocationOffset))
	//now, _ := time.ParseInLocation("2006-01-02 15:04:05", "2020-05-07 22:00:00",
	//	time.FixedZone(t.LocationName, t.LocationOffset))
	var (
		change = &change{}
		nextAt = &nextAt{}
	)
	//当前周期的上一个周期是否需要跳跃
	var jump = ""
	t.Recursive(now, change, nextAt, &jump)
	res := time.Date(nextAt.year, time.Month(nextAt.month), nextAt.day, nextAt.hour, nextAt.minute, 0, 0, now.Location())
	//fmt.Println("周", t.Dow)
	//fmt.Println("月", t.Month)
	//fmt.Println("日", t.Dom)
	//fmt.Println("时", t.Hour)
	//fmt.Println("分", t.Minute)
	fmt.Println(res.Format("2006-01-02 15:04:05"))
	if nextNum == 1 {
		return ""
	}
	return t.ComputeNextRunAt(res, nextNum-1)
}

//计算下一个日期
func (t *Task) Recursive(current time.Time, change *change, at *nextAt, jump *string) {
	t.nextMonth(current, change, at, jump)
	if b := t.nextDay(current, change, at, jump); !b {
		t.Recursive(current, change, at, jump)
	}
	if b := t.nextHour(current, change, at, jump); !b {
		t.Recursive(current, change, at, jump)
	}
	if b := t.nextMinute(current, change, at, jump); !b {
		t.Recursive(current, change, at, jump)
	}
}
