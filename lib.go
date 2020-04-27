package gcron

import "sort"

//获取某月下面有多少天
func getDayCountInMonth(year, month int) int {
	monthDay := [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	monthDayLeapYear := [12]int{31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if month > 12 {
		month = month - 12
	}
	if isLeapYear(year) {
		return monthDayLeapYear[month-1]
	}
	return monthDay[month-1]
}

//判断某年是不是闰年
func isLeapYear(year int) bool {
	if year%100 != 0 && year%4 == 0 {
		return true
	}
	if year%100 == 0 && year%400 == 0 {
		return true
	}
	return false
}

//判断一个元素是否在数组中
func existsInArray(arr []int, item int) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

//在数组中寻找比给定目标大的最小的元素
func getMinValueInArray(arr []int, item int) int {
	sort.Ints(arr)
	for _, v := range arr {
		if v > item {
			return v
		}
	}
	//如果目标是当前范围的最大值,就返回最小
	return arr[0]
}

//获取指定周中大于目标的最小元素
func getMinWeekInArray(arr []int, item int) int {
	sort.Ints(arr)
	//0 代表周末
	if item == 0 {
		if arr[0] == 0 {
			return arr[1]
		} else {
			return arr[0]
		}
	}
	for _, v := range arr {
		if v > item {
			return v
		}
	}
	return arr[0]
}
