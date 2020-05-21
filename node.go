package gcron

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/gomodule/redigo/redis"
	"log"
	"time"
)

//服务节点
type Node struct {
	EtcdClient *clientv3.Client
	//Host 用于rpc通讯
	Host string
	//租约id
	LeaseId clientv3.LeaseID
	//租约时间
	LeaseTTL   int64
	LeaderKey  string
	LeaderHost string
	IsLeader   bool
	NodePrefix string
	Ticker     *time.Ticker
	JobManager *JobManager
	//节点列表
	NodeList []string
}

//启动一个节点
//StartNewNode(os.Args[1], os.Args[2:])
//StartNewNode("localhost:9090", []string{"localhost:2379", "localhost:22379", "localhost:32379"})
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
		JobManager: NewJobManager(),
		NodeList:   make([]string, 0),
	}
	node.ApplyLeaseAndKeepAlive()
	node.RegisterEtcd()
	node.UpdateNodeList()
	node.ListenLeader()
	node.ListenNodeList()
	node.JobManager.Start()
}

//注册到etcd
func (node *Node) RegisterEtcd() {
	key := node.NodePrefix + node.Host
	value := fmt.Sprint(time.Now().Unix())
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	_, err := node.EtcdClient.Put(ctx, key, value, clientv3.WithLease(node.LeaseId))
	if err != nil {
		log.Fatalln(err.Error())
	}
}

//申请一个租约,并开启自动续租
func (node *Node) ApplyLeaseAndKeepAlive() {
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
func (node *Node) ListenLeader() {
	//当前不存在leader,立即开始竞选leader
	if node.ExistsLeader() == false {
		node.ElectLeader()
		return
	}
	node.GetLeaderHost()
	rch := node.EtcdClient.Watch(context.TODO(), node.LeaderKey)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			//监听到leader失效,开始竞选leader
			if ev.Type == mvccpb.DELETE {
				node.ElectLeader()
			}
		}
	}
}

//监听节点列表
func (node *Node) ListenNodeList() {
	rch := node.EtcdClient.Watch(context.TODO(), node.NodePrefix, clientv3.WithPrefix())
	for wresp := range rch {
		if len(wresp.Events) > 0 {
			node.UpdateNodeList()
		}
	}
}

//获取leader的host
func (node *Node) GetLeaderHost() {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.EtcdClient.Get(ctx, node.LeaderKey)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if len(resp.Kvs) > 0 {
		for _, value := range resp.Kvs {
			node.LeaderHost = string(value.Value)
		}
	}
}

//检查leader是否存在
func (node *Node) ExistsLeader() bool {
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

//竞选leader
func (node *Node) ElectLeader() {
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
		//不存在,继续发起竞选
		if node.ExistsLeader() == true {
			return
		} else {
			node.ElectLeader()
		}
	} else {
		//竞选成功,//将自己设置为leader
		_, err = node.EtcdClient.Put(ctx, node.LeaderKey, node.Host, clientv3.WithLease(node.LeaseId))
		if err == nil {
			node.IsLeader = true
			node.LeaderHost = node.Host
			node.Schedule()
			//停止任务执行
			node.JobManager.Stop <- true
		}
	}
}

//获取可用节点
func (node *Node) UpdateNodeList() {
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
	node.NodeList = list
}

//启动调度器(如果当前是leader)
//从 redis-zset中获取执行时间已到的任务放到 redis-set 中
func (node *Node) Schedule() {
	node.Ticker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-node.Ticker.C:
				//每一分钟触发一次任务调度,将需要执行的任务id放到Rpc服务的ReadyJobChan通道中,等待节点来取
				if time.Now().Second() == 0 {
					unix := time.Now().Unix()
					jobIds, err := redis.Strings(
						RedisInstance().Do("ZRANGEBYSCORE", RedisConfig.JobMeta, 0, unix),
					)
					if err != nil || len(jobIds) == 0 {
						continue
					}
					args := make([]interface{}, 1)
					args[0] = RedisConfig.Ready
					for _, jobId := range jobIds {
						args = append(args, jobId)
					}
					//将待执行任务放进 ready 集合
					RedisInstance().Do("SADD", args...)
				}
			}
		}
	}()
}
