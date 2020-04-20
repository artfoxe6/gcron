package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 0; i < 10; i++ {
		go func() {
			Print(i)
		}()
	}
	for i := 0; i < 10; i++ {
		go func() {
			Print(i)
		}()
	}
	time.Sleep(time.Second * 10)
}
func Print(num int) {
	time.Sleep(time.Second * time.Duration(num))
	fmt.Println(num)
}
