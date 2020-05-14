package gcron

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func startTestHttpServer(stop, start chan int) {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {

		headers := fmt.Sprint(request.Header) + "\r\n"

		if strings.ToUpper(request.Method) == "POST" {
			userId := request.PostFormValue("user_id")
			if userId == "" {
				_, _ = io.WriteString(writer, headers+"Post: homePage")
			} else {
				_, _ = io.WriteString(writer, headers+"Post User "+userId)
			}

		} else {
			userId := request.URL.Query().Get("user_id")
			if userId == "" {
				_, _ = io.WriteString(writer, headers+"Get: homePage")
			} else {
				_, _ = io.WriteString(writer, headers+"Get User:"+userId)
			}
		}

	})
	srv := &http.Server{Addr: ":1040"}
	go func() {
		start <- 1
		_ = srv.ListenAndServe()
	}()
	<-stop
	_ = srv.Shutdown(context.Background())
}

func TestSendHttp(t *testing.T) {
	t.Skip()
	fmt.Println("-------------TestSendHttp-------------")
	stop := make(chan int)
	start := make(chan int)
	go startTestHttpServer(stop, start)
	<-start
	job := CronJob{
		Args: map[string]interface{}{
			"user_id": "12",
		},
		Url:    "http://127.0.0.1:1040",
		Method: "post",
		Header: map[string]string{
			"Version": "1.0",
		},
		TTL: 100,
	}
	data, _ := json.Marshal(job)

	hd := httpData([]byte(data))
	body, _, err := hd.SendHttp()
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(string(body))
	stop <- 1
}
