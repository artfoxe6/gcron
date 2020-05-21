package main

import "fmt"

type Test struct {
	Close chan bool
}

func main() {
	args := make([]interface{}, 10)
	args[0] = "ttt"
	jobIds := []string{"1", "2"}
	for _, jobId := range jobIds {
		args = append(args, jobId)
	}
	fmt.Println(args)
}
