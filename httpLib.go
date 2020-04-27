package gcron

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type httpData []byte

//发送http请求
func (h httpData) SendHttp() ([]byte, error) {
	job := Job{}

	err := json.Unmarshal(h, &job)
	if err != nil {
		return nil, errors.New("任务格式解析错误")
	}
	if job.At == 0 || job.Url == "" || job.Method == "" {
		return nil, errors.New("参数至少包含 ExecTime,Method,Url")
	}
	return h.send(&job)
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
func (h *httpData) getClient(job *Job) (*http.Client, *http.Request, error) {
	params := parseArgs(job.Args)
	var request *http.Request
	var err error
	method := strings.ToUpper(job.Method)
	if method == "GET" {
		request, err = http.NewRequest(method, job.Url+"?"+params.Encode(), nil)
	} else {
		request, err = http.NewRequest(method, job.Url, strings.NewReader(params.Encode()))
	}
	if err != nil {
		return nil, nil, errors.New("create request error")
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
	return client, request, nil
}

func (h *httpData) send(job *Job) ([]byte, error) {
	client, request, err := h.getClient(job)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 200 {
		return body, nil
	}
	return nil, errors.New(strconv.Itoa(resp.StatusCode))
}
