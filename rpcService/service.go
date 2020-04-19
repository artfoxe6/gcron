package rpcService

import (
	"context"
	"gcron"
	"gcron/jobRPC"
	"google.golang.org/grpc"
	"log"
	"net"
)

type server struct {
	jobRPC.UnimplementedJobTimerServer
	JobManager *gcron.JobManager
	Stop       chan int
}

var RpcServer *server

func Run() {
	lis, err := net.Listen("tcp", ":1028")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	RpcServer = &server{
		JobManager: gcron.NewJobManager(),
		Stop:       make(chan int),
	}
	jobRPC.RegisterJobTimerServer(s, RpcServer)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("error:", err)
			}
		}()
		<-RpcServer.Stop
		s.GracefulStop()
	}()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func Stop() {
	if RpcServer != nil {
		RpcServer.Stop <- 1
	}
}

func (s *server) Add(ctx context.Context, in *jobRPC.Input) (*jobRPC.Response, error) {
	jobData := []byte(in.JsonData)
	s.JobManager.InBuffer <- jobData
	return &jobRPC.Response{Status: 200}, nil
}
