package main

import "gcron"

func main() {

	cm := gcron.NewCronManager()
	cm.Add("0 0 1,15 * 3", "123")
	//t := time.Date(2020, 9, 1, 0, 0, 0, 0, time.Local)
	//fmt.Println(t.Format("2006-01-02 15:04:05"))
	//t1 := time.Now()
	//t1.AddDate(0, 3, 0)
	//fmt.Println(t1.Year())

	//fmt.Println(0 | 0 | 0)
}
