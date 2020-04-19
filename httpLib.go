package gcron

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type httpData []byte

//发送http请求
func (h *httpData) SendHttp() {
	job := Job{}

	err := json.Unmarshal(*h, &job)
	if err != nil {
		return
	}

	if job.ExecTime == 0 || job.Url == "" || job.Method == "" {
		fmt.Println("参数至少包含 ExecTime,Method,Url")
		return
	}
	h.send(&job)
}

//构造参数
func parseArgs(args map[string]interface{}) url.Values {
	data := url.Values{}
	if args != nil {
		for key, v := range args {
			data.Add(key, fmt.Sprint(v))
		}
	}
	return data
}

//初始化client,request
func (h *httpData) getClient(job *Job) (*http.Client, *http.Request, bool) {
	params := parseArgs(job.Args)
	var request *http.Request
	var err error
	method := strings.Title(job.Method)
	if method == "GET" {
		request, err = http.NewRequest(method, job.Url+"?"+params.Encode(), nil)
	} else {
		request, err = http.NewRequest(method, job.Url, strings.NewReader(params.Encode()))
	}
	if err != nil {
		return nil, nil, false
	}
	if method == "POST" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if job.Header != nil {
		for k, v := range job.Header {
			request.Header.Set(k, v)
		}
	}
	client := &http.Client{
		Timeout: time.Second * 60,
	}
	return client, request, true
}

func (h *httpData) send(job *Job) (string, error) {
	client, request, b := h.getClient(job)
	if !b {
		return "", errors.New("get client error")
	}
	resp, err := client.Do(request)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	if resp.StatusCode == 200 {
		return string(body), nil
	}
	return "", errors.New(strconv.Itoa(resp.StatusCode))
}
