package main

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"log"
	"time"
)

func main() {
	//StartNewNode(os.Args[1], os.Args[2:])
	StartNewNode("localhost:9090", []string{"localhost:2379", "localhost:22379", "localhost:32379"})
}

type Node struct {
	EtcdClient *clientv3.Client
	//Host 用于rpc通讯
	Host       string
	LeaseId    clientv3.LeaseID
	LeaseTTL   int64
	LeaderKey  string
	IsLeader   bool
	NodePrefix string
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
		LeaseTTL:   1,
		NodePrefix: "/nodes/",
		LeaderKey:  "/leader",
	}
	//node.applyLeaseAndKeepAlive()
	//node.registerEtcd()
	node.listenLeader()
}

//注册到etcd
func (node *Node) registerEtcd() {
	key := node.NodePrefix + node.Host
	value := fmt.Sprint(time.Now().Unix())
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	_, err := node.EtcdClient.Put(ctx, key, value, clientv3.WithLease(node.LeaseId))
	if err != nil {
		log.Fatalln(err.Error())
	}
}

//申请一个租约,并开启自动续租
func (node *Node) applyLeaseAndKeepAlive() {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.EtcdClient.Grant(ctx, node.LeaseTTL)
	if err != nil {
		log.Fatalln(err.Error())
	}
	node.LeaseId = resp.ID
	ch, err := node.EtcdClient.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Fatalln(err.Error())
	}
	<-ch
}

//监听leader
func (node *Node) listenLeader() {
	//当前不存在leader,立即开始竞选leader
	if node.existsLeader() == false {
		node.electLeader()
		return
	}
	rch := node.EtcdClient.Watch(context.TODO(), node.LeaderKey)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			//监听到leader失效,开始竞选leader
			if ev.Type == mvccpb.DELETE {
				node.electLeader()
			}
		}
	}
}

//检查leader是否存在
func (node *Node) existsLeader() bool {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.EtcdClient.Get(ctx, node.LeaderKey)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if len(resp.Kvs) == 0 {
		return false
	}
	return true
}

//竞争leader
func (node *Node) electLeader() {
	s, err := concurrency.NewSession(node.EtcdClient)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	e := concurrency.NewElection(s, node.LeaderKey+"/")
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err = e.Campaign(ctx, node.Host); err != nil {
		//竞争失败或超时
		//检查当前是否存在leader,如果已存在,就放弃竞选
		//不存在,继续发起竞争
		if node.existsLeader() == true {
			return
		} else {
			node.electLeader()
		}
	}
	//成为leader
	node.IsLeader = true
}

//获取可用节点
func (node *Node) nodeList() []string {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.EtcdClient.Get(ctx, node.NodePrefix, clientv3.WithPrefix())
	if err != nil {
		log.Fatalln(err.Error())
	}
	list := make([]string, 0)
	for _, v := range resp.Kvs {
		//value值规则 /nodes/127.0.0.1:3456
		//去掉前缀,只保留host
		host := string(v.Key)[7:]
		if host != node.Host {
			list = append(list, host)
		}
	}
	return list
}

//开启rpc
func (node *Node) startRpc() {

}
