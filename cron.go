package gcron

import (
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

//linux下crontab的golang实现,不支持单个标志位的复合写法 比如 2-8/2, 请使用2,4,6,8代替
// 分、时、日、月、周
// * 取值范围内的所有数字
// / 每过多少个数字
// - 从X到Z
// ，散列数字

type Task struct {
	Second    Rule       //秒 规则
	Minute    Rule       //分 规则
	Hour      Rule       //时 规则
	Day       Rule       //日 规则
	Month     Rule       //月 规则
	Week      Rule       //周 规则
	JoinAt    time.Time  //任务添加时间
	LastRunAt *time.Time //上次执行时间
	NextRunAt *time.Time //执行时间
	ExecTimes int        //执行次数
	TaskData  TaskData   //任务数据
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

//任务规则
type Rule struct {
	RunType RunType
	Value   []int
}
type RunType int //周期类型,不直接针对任务,而是任务下面的时分秒
const (
	Every    RunType = iota //范围内所有数字
	Assign                  //指定具体数字
	Interval                //每间隔多久执行一次
)

//任务管理
type CronManager struct {
	Tasks    []*Task
	Location *time.Location
}

//创建任务管理器
func NewCronManager(location *time.Location) *CronManager {
	return &CronManager{
		Tasks:    make([]*Task, 100),
		Location: location,
	}
}

//添加任务
func (cm *CronManager) Add(second, minute, hour, day, month, week, taskJson string) {
	taskData := TaskData{}
	err := json.Unmarshal([]byte(taskJson), &taskData)
	if err != nil {
		log.Fatalln("数据格式错误")
	}
	task := &Task{
		Second:    parse(second),
		Minute:    parse(minute),
		Hour:      parse(hour),
		Day:       parse(day),
		Month:     parse(month),
		Week:      parse(week),
		JoinAt:    time.Now(),
		LastRunAt: nil,
		NextRunAt: nil,
		ExecTimes: 0,
		TaskData:  taskData,
	}
	task.ComputeNextRunAt()
	cm.Tasks = append(cm.Tasks, task)
}

//规则解析
func parse(param string) Rule {
	if ok := strings.ContainsRune(param, '/'); ok {
		arr := strings.Split(param, "/")
		value, _ := strconv.Atoi(arr[1])
		return Rule{RunType: Interval, Value: []int{value}}
	}
	if ok := strings.ContainsRune(param, '-'); ok {
		arr := strings.Split(param, "-")
		value := make([]int, 31)
		begin, _ := strconv.Atoi(arr[0])
		end, _ := strconv.Atoi(arr[1])
		for i := begin; i <= end; i++ {
			value = append(value, i)
		}
		sort.Ints(value)
		return Rule{RunType: Assign, Value: value}
	}
	if ok := strings.ContainsRune(param, ','); ok {
		arr := strings.Split(param, ",")
		value := make([]int, 31)
		for i := 0; i <= len(arr); i++ {
			tmp, _ := strconv.Atoi(arr[i])
			value = append(value, tmp)
		}
		sort.Ints(value)
		return Rule{RunType: Assign, Value: value}
	}
	return Rule{RunType: Every, Value: nil}
}

//任务列表
func (cm *CronManager) List() []*Task {
	return cm.Tasks
}

//时间规则范围
var (
	seconds = [2]int{0, 59}
	minutes = [2]int{0, 59}
	hours   = [2]int{0, 23}
	day     = [2]int{1, 31}
	months  = [2]int{1, 12}
	week    = [2]int{0, 6}
)

func (t *Task) ComputeNextRunAt() {
	//second, minute, hour, day, month, week
	//"*/10", "*/2",  "*",  "*", "*/3", "*"
	//2020-04-27 14:16:30
	now := time.Now()
	nextRunAt := time.Now()
	var (
		month = int(now.Month())
		//day    = now.Day()
		//hour   = now.Hour()
		//minute = now.Minute()
		//second = now.Second()
		week = int(now.Weekday())
	)
	//处理 Month 规则
	switch t.Month.RunType {
	case Assign:
		//[1,2,3,6,8] current:4
		//当月不再指定月份中,选择最近的一个月份
		if !existsInArray(t.Month.Value, month) {
			targetMonth := getMinValueInArray(t.Month.Value, month)
			nextRunAt = nextRunAt.AddDate(0, targetMonth-month, 0)
		} else {
			//todo 如果当月在指定月份中,不做处理,但是需要条件回查
		}
	case Interval:
		if t.LastRunAt == nil {
			nextRunAt = nextRunAt.AddDate(0, t.Month.Value[0], 0)
		} else {
			lastMonthAt := int(t.LastRunAt.Month())
			targetMonth := lastMonthAt + t.Month.Value[0] - month
			nextRunAt = nextRunAt.AddDate(0, targetMonth, 0)
		}
	}
	//处理 Week 规则
	switch t.Week.RunType {
	case Assign:
		//[3,6,0] current:1
		//当周不再指定周中,那么选择最近的一个周
		if !existsInArray(t.Week.Value, week) {
			targetWeek := getMinWeekInArray(t.Week.Value, week)
			addDay := 0
			if targetWeek != week {
				if targetWeek == 0 {
					addDay = 7 - week
				} else {
					addDay = targetWeek - week
				}
			}
			nextRunAt = nextRunAt.AddDate(0, 0, addDay)
		} else {
			//todo 如果当周在指定周中,不做处理,但是需要条件回查
		}
	case Interval:
		//还没有执行过
		if t.LastRunAt == nil {
			nextRunAt = nextRunAt.AddDate(0, 0, t.Week.Value[0]*7)
		} else {
			//temp := t.LastRunAt.AddDate(0, 0, t.Week.Value[0]*7)
			//addDay := (int(temp.Unix()) - seconds) / 86400
			//targetWeek := lastWeekAt + t.Week.Value[0] - week
			//nextRunAt = nextRunAt.AddDate(0, targetMonth, 0)
		}
	}
}
