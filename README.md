# gcron

分布式定时任务系统


leader调度,将所有将要执行的任务放进等待队列,
worker通过rpc不停的发送请求获取任务执行