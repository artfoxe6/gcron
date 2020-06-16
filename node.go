package gcron

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/gomodule/redigo/redis"
	"log"
	"strconv"
	"time"
)

//服务节点
type Node struct {
	ETCDClient      *clientv3.Client //etcd客户端
	Host            string           //Host 用于rpc通讯
	LeaseId         clientv3.LeaseID //租约id
	LeaseTTL        int64            //租约时间
	LeaderKey       string           //Leader Key
	NodePrefix      string           //Node Prefix
	Ticker          *time.Ticker     //Ticker
	JobManager      *JobManager      //任务管理器
	NodeList        []string         //节点列表
	Scheduling      bool             //调度器状态
	Close           chan bool        //关闭节点
	SingleNodeModel bool             //单机模式
}

//启动一个节点
func StartNewNode(host string, etcdNodes []string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   etcdNodes,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalln("无法连接到ETCD服务")
	}
	node := Node{
		ETCDClient:      cli,
		Host:            host,
		LeaseTTL:        1,
		NodePrefix:      "/nodes/",
		LeaderKey:       "/leader",
		JobManager:      NewJobManager(),
		NodeList:        make([]string, 0),
		Close:           make(chan bool, 0),
		Scheduling:      false,
		SingleNodeModel: false,
	}
	//申请一个租约,并开启自动续租
	node.ApplyLeaseAndKeepAlive()
	//注册到etcd
	node.RegisterEtcd()
	//保持健康检测
	node.KeepHealthy()
	//监听leader,如果不存在leader,参与leader竞选
	node.WatchLeader()
	//观察节点变化
	node.WatchNodeList()
	//node.JobManager.Start()
	<-node.Close
}

//注册到etcd
func (node *Node) RegisterEtcd() {
	fmt.Println("注册到ETCD")
	key := node.NodePrefix + node.Host
	value := time.Now().Format("2006-01-02 15:04:05")
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.ETCDClient.Put(ctx, key, value, clientv3.WithLease(node.LeaseId))
	if err != nil {
		log.Fatalln(err.Error())
	}
	fmt.Println("RevisionID:" + strconv.Itoa(int(resp.Header.Revision)))

}

//申请一个租约,并开启自动续租
func (node *Node) ApplyLeaseAndKeepAlive() {
	fmt.Println("申请租约")
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.ETCDClient.Grant(ctx, node.LeaseTTL)
	if err != nil {
		log.Fatalln(err.Error())
	}
	node.LeaseId = resp.ID
	ch, err := node.ETCDClient.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Fatalln(err.Error())
	}
	<-ch
}

//监听leader
func (node *Node) WatchLeader() {
	fmt.Println("监听Leader节点")
	//node.GetLeaderHost()
	go func() {
		rch := node.ETCDClient.Watch(context.TODO(), node.LeaderKey, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				fmt.Println("Leader节点发生变化:", ev.Type)
				//监听到新Leader
				if ev.Type == mvccpb.PUT {
					if node.GetLeaderHost() != node.Host {
						//停止调度
						node.Scheduling = false
						//启动任务执行
						go node.JobManager.Start()
					} else {
						//启动调度器
						node.Schedule()
						//停止任务执行
						go node.JobManager.Stop()
						//检查是否启动单节点模式
						node.CheckSingleNodeModel()
					}
				}
				//监听到leader失效,开始竞选leader
				if ev.Type == mvccpb.DELETE {
					go node.ElectLeader()
				}
			}
		}

	}()
	//当前不存在leader,立即开始竞选leader
	if node.ExistsLeader() == false {
		node.ElectLeader()
	} else {
		//存在Leader,启动任务处理
		node.JobManager.Start()
	}

}

//监听节点列表
func (node *Node) WatchNodeList() {
	rch := node.ETCDClient.Watch(context.TODO(), node.NodePrefix, clientv3.WithPrefix())
	for wresp := range rch {
		if len(wresp.Events) > 0 {
			fmt.Println("node列表发生变化")
			//非Leader无需处理节点变化
			if node.Scheduling == false {
				fmt.Println("不处理")
				continue
			}
			node.CheckSingleNodeModel()
		}
	}
}

//检查节点列表
func (node *Node) CheckSingleNodeModel() {
	node.UpdateNodeList()
	fmt.Println(node.NodeList)
	if len(node.NodeList) == 0 {
		//切换到单机模式
		node.SingleNodeModel = true
		fmt.Println("未检测到worker节点，启动单节点模式")
		node.JobManager.Start()
	} else {
		if node.SingleNodeModel {
			node.SingleNodeModel = false
			fmt.Println("退出单节点模式")
			node.JobManager.Stop()
		}
	}
}

//获取leader的host
func (node *Node) GetLeaderHost() string {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.ETCDClient.Get(ctx, node.LeaderKey)
	if err != nil {
		log.Fatalln(err.Error())
	}
	leaderHost := ""
	if len(resp.Kvs) > 0 {
		for _, value := range resp.Kvs {
			leaderHost = string(value.Value)
		}
	}
	return leaderHost
}

//检查leader是否存在
func (node *Node) ExistsLeader() bool {
	fmt.Print("检查Leader是否存在")
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.ETCDClient.Get(ctx, node.LeaderKey)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if len(resp.Kvs) == 0 {
		fmt.Println(" - 不存在")
		return false
	}
	fmt.Println(" - 存在")
	return true
}

//竞选leader
func (node *Node) ElectLeader() {
	fmt.Println("开始竞选Leader")
	s, err := concurrency.NewSession(node.ETCDClient)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	e := concurrency.NewElection(s, "/election")
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	if err = e.Campaign(context.TODO(), node.Host); err != nil {
		//竞争失败或超时
		//检查当前是否存在leader,如果已存在,就放弃竞选
		//不存在,继续发起竞选
		fmt.Println("竞选错误")
		if node.ExistsLeader() == true {
			fmt.Println("已经存在Leader")
			return
		} else {
			fmt.Println("未检测到Leader，再次竞选")
			node.ElectLeader()
		}
	} else {
		fmt.Println("竞选结束")
		leader, err := e.Leader(ctx)
		if err != nil {
			fmt.Printf("Leader() returned non nil err: %s", err)
		}
		if string(leader.Kvs[0].Value) == node.Host && node.ExistsLeader() == false {
			fmt.Println("竞选成功:" + string(leader.Kvs[0].Value))
			//竞选成功,//将自己设置为leader
			_, err = node.ETCDClient.Put(ctx, node.LeaderKey, node.Host, clientv3.WithLease(node.LeaseId))
		} else {
			fmt.Println("竞选失败")
		}
	}
}

//获取可用节点
func (node *Node) UpdateNodeList() {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := node.ETCDClient.Get(ctx, node.NodePrefix, clientv3.WithPrefix())
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
	fmt.Println("启动任务调度器")
	node.Scheduling = true
	node.Ticker = time.NewTicker(time.Second)
	go func() {
		for {
			if node.Scheduling == false {
				fmt.Println("任务调度器停止")
				return
			}
			select {
			case <-node.Ticker.C:
				fmt.Println("调度" + time.Now().Format("2006-01-02 15:04:05"))
				//每一分钟触发一次任务调度,将需要执行的任务id放到Rpc服务的ReadyJobChan通道中,等待节点来取
				if time.Now().Second() == 0 {
					unix := time.Now().Unix()
					jobIds, err := redis.Strings(
						RedisInstance().Do("ZRANGEBYSCORE", RedisConfig.JobMeta, 0, unix),
					)
					if err != nil || len(jobIds) == 0 {
						continue
					}
					//将待执行任务放进 ready 集合
					args := make([]interface{}, 1)
					args[0] = RedisConfig.Ready
					for _, jobId := range jobIds {
						args = append(args, jobId)
					}
					_, _ = RedisInstance().Do("SADD", args...)
				}
			}
		}
	}()
}

//保持健康检测，断开重联，每30s检测一次是否断开 Etcd Server
func (node *Node) KeepHealthy() {
	go func() {
		ticker := time.NewTicker(time.Second * 30)
		for {
			select {
			case <-ticker.C:
				fmt.Println("健康检测")
				key := node.NodePrefix + node.Host
				ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
				resp, err := node.ETCDClient.Get(ctx, key)
				if err == nil {
					//断开重联
					if len(resp.Kvs) == 0 {
						fmt.Println("正在断开重联")
						go node.RegisterEtcd()
					}
				}
			default:
				time.Sleep(time.Second * 5)
			}
		}
	}()
}
