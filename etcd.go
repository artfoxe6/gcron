package gcron

import (
	"go.etcd.io/etcd/clientv3"
	"log"
	"strings"
	"time"
)

var etcdInstance = &clientv3.Client{}
var etcdIsLoad = false

func connection() *clientv3.Client {
	var err error
	etcdInstance, err = clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(EtcdConfig.Host, ","),
		DialTimeout: time.Duration(EtcdConfig.Timeout) * time.Second,
	})
	if err != nil {
		log.Fatalln("etcd 连接失败 " + err.Error())
	}
	etcdIsLoad = true
	return etcdInstance
}

func EtcdInstance() *clientv3.Client {
	if !etcdIsLoad {
		connection()
	}
	return etcdInstance
}
