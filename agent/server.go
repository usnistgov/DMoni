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
	s.ag.RLock()
	// check if already registered
	if _, ok := s.ag.apps[in.Id]; ok {
		s.ag.RUnlock()
		return &pb.RegReply{}, nil
	}
	s.ag.RUnlock()

	//grpclog.Printf("Register app %s", in.Id)
	grpclog.Printf("Register app %s, entry pid %d", in.Id, in.Pid)

	app := &common.App{
		Id:         in.Id,
		Frameworks: in.Frameworks,
		JobIds:     in.JobIds,
		EntryPid:   in.Pid,
	}
	s.ag.Lock()
	s.ag.apps[app.Id] = app
	s.ag.appProcs[app.Id] = make([]common.Process, 0)
	s.ag.Unlock()

	return &pb.RegReply{}, nil
}

// Application deregistration service
func (s *agentServer) Deregister(ctx context.Context, in *pb.DeregRequest) (*pb.DeregReply, error) {
	// Check if the application exists
	s.ag.RLock()
	if _, ok := s.ag.apps[in.AppId]; !ok {
		s.ag.RUnlock()
		return nil, nil
	}
	s.ag.RUnlock()

	grpclog.Printf("Deregister app %s", in.AppId)

	s.ag.Lock()
	delete(s.ag.apps, in.AppId)
	delete(s.ag.appProcs, in.AppId)
	s.ag.Unlock()
	return &pb.DeregReply{}, nil
}

// Obtaining Applications' processes
func (s *agentServer) GetProcesses(ctx context.Context, in *pb.ProcRequest) (*pb.ProcList, error) {
	s.ag.RLock()
	// check if application exists
	if _, ok := s.ag.apps[in.AppId]; !ok {
		s.ag.RUnlock()
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.AppId))
	}
	s.ag.RUnlock()

	grpclog.Printf("GetProcesses of app %s", in.AppId)

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
