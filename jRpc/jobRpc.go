package jRpc

import (
	"io"
)

type RpcServer struct {
	UnimplementedJobTransferServer
	WaitJobQueue chan int64
}

//传输任务,由leader节点分发给node节点
func (s *RpcServer) Transfer(pipe JobTransfer_TransferServer) error {
	for {
		_, err := pipe.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		jobId := <-s.WaitJobQueue
		err = pipe.Send(&Response{JobId: jobId})
		//如果发送失败,任务重新回到等待通道
		if err != nil {
			s.WaitJobQueue <- jobId
		}
	}
}
