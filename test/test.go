package main

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"log"
	"time"
)

func main() {
	//StartNewNode(os.Args[1], os.Args[2:])
	StartNewNode("localhost:9090", []string{"localhost:2379", "localhost:22379", "localhost:32379"})
}

type Node struct {
	EtcdClient *clientv3.Client
	Host       string
}

//启动一个节点
func StartNewNode(host string, etcdNodes []string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdNodes,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalln(err.Error())
	}
	node := Node{
		EtcdClient: cli,
		Host:       host,
	}
	node.applyLease()
}

//注册到etcd
func (node *Node) registerEtcd() {}

//申请租约
func (node *Node) applyLease() {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.EtcdClient.Get(ctx, "foo")
	if err != nil {
		log.Fatalln(err.Error())
	}
	for _, ev := range resp.Kvs {
		fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	}
}

//续租
func (node *Node) updateLease() {}

//监听master
func (node *Node) listenMaster() {}

//竞选master
func (node *Node) electMaster() {}

//开启rpc
func (node *Node) startRpc() {}
