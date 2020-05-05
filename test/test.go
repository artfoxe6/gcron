package main

import (
	"gcron"
)

func main() {
	cm := gcron.NewCronManager()
	cm.Add("* */2 * */2 *", "123")

	//t1 := time.Now()
	//t1.AddDate(0, 3, 0)
	//fmt.Println(t1.Year())

	//fmt.Println(0 | 0 | 0)
}
