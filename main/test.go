package main

import (
	"fmt"
	"time"
)

type Test struct {
	Close chan bool
}

func main() {
	t := &Test{Close: make(chan bool, 0)}
	t.echo("msg")
	time.Sleep(time.Second * 3)
	t.Close <- true
	go func() {
		time.Sleep(time.Second * 3)
		<-t.Close
	}()
	t.Close <- true
	time.Sleep(time.Second * 12)
}

func (t *Test) echo(msg string) {
	go func() {
		for {
			select {
			case <-t.Close:
				go t.echo("submsg")
				return
			default:
				fmt.Println(msg)
			}
			time.Sleep(time.Second)
		}
	}()
	return
}
