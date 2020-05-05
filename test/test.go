package main

import (
	"fmt"
	"gcron"
)

func main() {
	cm := gcron.NewCronManager()
	cm.Add("0 0,12 1 */2 *", "123")
	fmt.Println("2020-07-01 00:00:00")

	//t1 := time.Now()
	//t1.AddDate(0, 3, 0)
	//fmt.Println(t1.Year())

	//fmt.Println(0 | 0 | 0)
}
