# gcron

Go语言实现的分布式高可用定时任务系统 <br />

# 关系说明
本系统拆分为两个独立的项目：
1. 管理面板 [gcron-api](https://github.com/artfoxe6/gcron-api) <br />
>负责定时任务的增删改查，执行日志查看，和调度执行服务通讯
2. 调度和任务系统
>由一个调度节点（以下称作Leader节点）和若干个任务节点（Worker节点）组成 <br />
>Leader节点和Worker节点启动无任何区别，节点间通过竞选确定谁作Leader节点，<br />
>Leader节点挂掉后会自动重选<br />

# 系统架构
## 简明架构流程
### 整体流程
>通过管理面板添加定时任务，当然不仅仅能通过面板添加，面板系统还提供http和rpc方式添加任务<br />
>任务的添加修改删除，会更新到MySQL，并同步到Redis <br/>
>启动调度和任务系统，调度器将需要执行的任务发送到准备队列中，<br/>
>任务节点从准备队列中拉取任务并执行<br/>

### 节点启动流程
>节点启动后首先注册到etcd，探测Leader是否存在，如果不存在，发起竞选，<br />
>有Leader或竞选失败，当前节点将自动切换为Worker角色，开始拉取和执行任务，<br />
>如果竞选成功，当前节点自动切换为Leader角色，开始调度任务，并停止任务拉取工作<br />

### 关键组件
>系统中有两个关键的组件，redis和etcd<br/>
>redis负责任务的流转，管理面板和调度系统之间流转，调度器和任务节点间流转<br />
>etcd负责节点的自动发现和Leader竞选，是整个系统高可用的关键<br />
                                                                                                                                                                                                   
      
### 架构和节点启动流程图
**如无法查看，请clone项目到本地，存在/images下**
 <br />![系统架构图](https://raw.githubusercontent.com/artfoxe6/gcron/master/images/system.png)
 <br />![节点启动流程图](https://raw.githubusercontent.com/artfoxe6/gcron/master/images/node.png)

# 启动

go run main/main.go 127.0.0.1:9001 27.0.0.1:2379 27.0.0.1:22379 27.0.0.1:32379

