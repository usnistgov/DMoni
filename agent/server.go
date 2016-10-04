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

// Kill a launched application
func (s *agentServer) Kill(ctx context.Context, in *pb.KRequest) (*pb.KReply, error) {
	grpclog.Printf("Kill application %s", in.AppId)

	app := s.ag.getLchApp(in.AppId)
	if app == nil {
		return nil, errors.New("App does not exist")
	}

	s.ag.kill(app)
	return &pb.KReply{}, nil
}

// Register starts to collect resource usages for a given application.
func (s *agentServer) Register(ctx context.Context, in *pb.AppInfo) (*pb.RegReply, error) {
	grpclog.Printf("Register app %s", in.Id)

	app := s.ag.getMoniApp(in.Id)
	if app != nil {
		return nil, errors.New("App has already registered")
	}

	app = &App{
		Id:         in.Id,
		Frameworks: in.Frameworks,
		JobIds:     in.JobIds,
		Procs:      make([]common.Process, 0),
		ofile:      path.Join(outDir, in.Id),
	}
	s.ag.apps.Lock()
	s.ag.apps.m[app.Id] = app
	s.ag.apps.Unlock()

	return &pb.RegReply{}, nil
}

// Deregister stops collecting resource usage info for a given application,
// and triggers storing collected info in database.
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
	if in.Save == true {
		go func() {
			err := s.ag.storeData(a)
			if err != nil {
				log.Printf("Failed to store app %s data: %v", a.Id, err)
			}
		}()
	}

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
