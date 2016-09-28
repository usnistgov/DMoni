package agent

import (
	"errors"
	"fmt"
	"log"
	"net"
	"path"

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

// Launch starts an application
func (s *agentServer) Launch(ctx context.Context, in *pb.LchRequest) (*pb.LchReply, error) {
	grpclog.Printf("Launch application %s", in.AppId)
	err := s.ag.launch(in.AppId, in.ExecName, in.ExecArgs...)
	if err != nil {
		log.Printf("Failed to launch app %s: %v", in.AppId, err)
		return nil, err
	}
	return &pb.LchReply{}, nil
}

// Application registration service
func (s *agentServer) Register(ctx context.Context, in *pb.AppInfo) (*pb.RegReply, error) {
	apps := s.ag.apps
	apps.RLock()
	// check if already registered
	if _, ok := apps.m[in.Id]; ok {
		apps.RUnlock()
		return &pb.RegReply{}, nil
	}
	apps.RUnlock()

	//grpclog.Printf("Register app %s", in.Id)
	grpclog.Printf("Register app %s", in.Id)

	app := &App{
		Id:         in.Id,
		Frameworks: in.Frameworks,
		JobIds:     in.JobIds,
		Procs:      make([]common.Process, 0),
		ofile:      path.Join(outDir, in.Id),
	}
	apps.Lock()
	apps.m[app.Id] = app
	apps.Unlock()

	return &pb.RegReply{}, nil
}

// Application deregistration service
func (s *agentServer) Deregister(ctx context.Context, in *pb.DeregRequest) (*pb.DeregReply, error) {
	grpclog.Printf("Deregister app %s", in.AppId)

	apps := s.ag.apps
	// Check if the application exists
	apps.Lock()
	a, ok := apps.m[in.AppId]
	if !ok {
		apps.Unlock()
		return nil, nil
	}
	delete(apps.m, in.AppId)
	apps.Unlock()

	// Store app's perfromance data
	go func() {
		err := s.ag.storeData(a)
		if err != nil {
			log.Printf("Failed to store app %s data: %v", a.Id, err)
		}
	}()

	return &pb.DeregReply{}, nil
}

// Obtaining Applications' processes
func (s *agentServer) GetProcesses(ctx context.Context, in *pb.ProcRequest) (*pb.ProcList, error) {
	apps := s.ag.apps
	apps.RLock()
	// check if application exists
	a, ok := apps.m[in.AppId]
	if !ok {
		apps.RUnlock()
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.AppId))
	}
	apps.RUnlock()
	grpclog.Printf("GetProcesses of app %s", in.AppId)

	// Make a process list and Return it
	list := &pb.ProcList{
		Procs: make([]*pb.Process, len(a.Procs)),
	}
	for i, p := range a.Procs {
		list.Procs[i] = &pb.Process{
			Pid:  int64(p.Pid),
			Name: p.ShortName,
			Cmd:  p.FullName,
		}
	}
	return list, nil
}
