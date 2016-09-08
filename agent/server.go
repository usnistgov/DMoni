package agent

import (
	"errors"
	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	pb "github.com/lizhongz/dmoni/proto/agent"
)

type agentServer struct {
	ag *Agent
}

func newServer(ag *Agent) *agentServer {
	s := &agentServer{
		ag: ag,
	}
	return s
}

// Create grpc server and listen to connections
func (s *agentServer) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.ag.me.Port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterMonitorProcsServer(grpcServer, s)
	grpcServer.Serve(lis)
}

// Application registration service
func (s *agentServer) Register(ctx context.Context, in *pb.AppInfo) (*pb.RegReply, error) {
	// check if already registered
	if _, ok := s.ag.apps[in.Id]; ok {
		return &pb.RegReply{}, errors.New(fmt.Sprintf("App %s already registered", in.Id))
	}

	app := &common.App{
		Id:         in.Id,
		Frameworks: make([]string, len(in.Frameworks)),
	}
	copy(app.Frameworks, in.Frameworks)
	s.ag.apps[app.Id] = app
	s.ag.appProcs[app.Id] = make([]common.Process, 0)

	return &pb.RegReply{}, nil
}

// Application deregistration service
func (s *agentServer) Deregister(ctx context.Context, in *pb.DeregRequest) (*pb.DeregReply, error) {

	// Check if the application exists
	if _, ok := s.ag.apps[in.AppId]; !ok {
		return nil, nil
	}

	delete(s.ag.apps, in.AppId)
	delete(s.ag.appProcs, in.AppId)
	return &pb.DeregReply{}, nil
}

// Obtaining Applications' processes
func (s *agentServer) GetProcesses(ctx context.Context, in *pb.ProcRequest) (*pb.ProcList, error) {

	// check if application exists
	if _, ok := s.ag.apps[in.AppId]; !ok {
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.AppId))
	}

	// Make a process list and Return it
	list := &pb.ProcList{
		Procs: make([]*pb.Process, len(s.ag.appProcs[in.AppId])),
	}
	for i, p := range s.ag.appProcs[in.AppId] {
		list.Procs[i] = &pb.Process{
			Pid:  p.Pid,
			Name: p.ShortName,
			Cmd:  p.FullName,
		}
	}
	return list, nil
}
