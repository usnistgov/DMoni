package agent

import (
	"errors"
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

// Application registration service
func (s *agentServer) Register(ctx context.Context, in *pb.AppInfo, opts ...grpc.CallOption) (*pb.RegReply, error) {
	// check if already registered
	if _, ok := s.ag.apps[in.Id]; ok {
		return &RegReply{}, errors.New(fmt.Sprintf("App %s already registered", in.Id))
	}

	app := &common.App{
		Id:         in.Id,
		Frameworks: make([]string, len(in.Frameworks)),
	}
	copy(app.Frameworks, in.Frameworks)
	s.ag.apps[app.Id] = app
	s.ag.appProcs[app.Id] = make([]common.Process)

	return &pb.RegReply{}, nil
}

// Application deregistration service
func (s *agentServer) Deregister(ctx context.Context, in *pb.DeregRequest, opts ...grpc.CallOption) (*pb.DeregReply, error) {

	// Check if the application exists
	if _, ok := s.ag.apps[In.AppId]; !ok {
		return nil, nil
	}

	delete(s.ag.apps, in.AppId)
	delete(s.ag.appsProcs, in.AppId)
	return &pb.DeregReply{}, nil
}

// Obtaining Applications' process service
func (s *agentServer) GetProcesses(ctx context.Context, in *pb.ProcRequest, opts ...grpc.CallOption) (*pb.ProcList, error) {

	// check if application exists
	if _, ok := s.ag.apps[app.Id]; !ok {
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.AppId))
	}

	// Make a process list and Return it
	list := &pb.ProcList{
		Procs: make([]pb.Process, len(s.ag.appProcs[in.AppId])),
	}
	for i, p := range s.ag.appProcs[in.AppId] {
		list[i].Pid = p.Pid
		list[i].Name = p.ShortName
		list[i].Cmd = p.FullName
	}
	return list, nil
}
