# gcron

Go语言实现的分布式高可用定时任务系统 <br />

#关系说明
本系统拆分为两个独立的项目：
1. 管理面板 [gcron-api](https://github.com/artfoxe6/gcron-api) <br />
    >负责定时任务的增删改查，执行日志查看，和调度执行服务通讯
2. 调度和任务系统
    >单个调度节点和若干个任务节点 <br />
    调度节点和任务节点启动无任何区别，节点间通过竞选选出调度节点，
    调度节点挂掉后会自动重选

# 系统架构
1. 简明架构流程
    >通过管理面板添加定时任务，当然不仅仅能通过面板添加，面板系统还提供http和rpc方式添加任务<br />
     任务的添加修改删除，会更新到MySQL，并同步到Redis <br/>
     启动调度和任务系统，调度器将需要执行的任务放到ready队列中，<br/>
     任务节点从ready中拉取任务并执行<br/>
    
    > Redis在其中起到关键的作用，<br/>
      
2. 架构图（未FQ无法查看，请下载项目到本地，在目录images下）：
 ![系统架构图](https://raw.githubusercontent.com/artfoxe6/gcron/master/images/system.png)
# 节点启动流程图
 ![节点启动流程图](https://raw.githubusercontent.com/artfoxe6/gcron/master/images/node.png)