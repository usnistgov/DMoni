package manager

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/satori/go.uuid"
	//"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	agPb "github.com/lizhongz/dmoni/proto/agent"
	mPb "github.com/lizhongz/dmoni/proto/manager"
)

type appServer struct {
	mng *Manager
}

func newAppServer(mng *Manager) *appServer {
	s := new(appServer)
	s.mng = mng
	return s
}

func (s *appServer) Run() {
	// Run application server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.mng.appPort))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	mPb.RegisterAppGaugeServer(grpcServer, s)
	grpcServer.Serve(lis)
}

// Register an application
func (s *appServer) Register(ctx context.Context, in *mPb.AppDesc) (*mPb.AppIndex, error) {
	// Generate an id for the application
	id := strings.Join(strings.Split(uuid.NewV4().String(), "-"), "")

	// Register the the application on manager
	m := s.mng
	app := &common.App{
		Id:         id,
		Frameworks: make([]string, len(in.Frameworks)),
	}
	copy(app.Frameworks, in.Frameworks)
	m.apps[id] = app

	// Register the app with all agents
	for _, ag := range m.agents {
		// Create a grpc client
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(fmt.Sprintf(
			"%s:%d", ag.Ip, ag.Port), opts...)
		if err != nil {
			grpclog.Printf("Failed to dial agent: %v", err)
			continue
		}
		defer conn.Close()
		client := agPb.NewMonitorProcsClient(conn)

		// Send app info to agent

		ai := &agPb.AppInfo{
			Id:         app.Id,
			Frameworks: make([]string, len(app.Frameworks)),
		}
		copy(ai.Frameworks, app.Frameworks)

		grpclog.Printf("Register app %s with agent %s", app.Id, ag.Id)
		_, err = client.Register(context.Background(), ai)
		if err != nil {
			grpclog.Printf("%v.Register(_) = _, %v", client, err)
			continue
		}
	}

	// TODO(lizhong): if failed to propagate app info to agents

	return &mPb.AppIndex{Id: id}, nil
}

// Deregister an app
func (s *appServer) Deregister(ctx context.Context, in *mPb.AppIndex) (*mPb.DeregReply, error) {
	if _, ok := s.mng.apps[in.Id]; !ok {
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.Id))
	}

	// Deregister the app with all agents
	m := s.mng
	for _, ag := range m.agents {
		// Create a grpc client to an agent
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(fmt.Sprintf(
			"%s:%d", ag.Ip, ag.Port), opts...)
		if err != nil {
			grpclog.Printf("Failed to dial agent: %v", err)
		}
		defer conn.Close()
		client := agPb.NewMonitorProcsClient(conn)

		// Degregister the app on the agent
		grpclog.Printf("Deregister app %s on with agent %s", in.Id, ag.Id)
		_, err = client.Deregister(context.Background(),
			&agPb.DeregRequest{AppId: in.Id})
		if err != nil {
			grpclog.Printf("%v.Deregister(_) = _, %v", client, err)
			continue
		}
		grpclog.Printf("Done Deregister app")
	}

	delete(m.apps, in.Id)
	return &mPb.DeregReply{}, nil
}

// Get the running status of an app
func (s *appServer) GetStatus(ctx context.Context, in *mPb.AppIndex) (*mPb.AppStatus, error) {
	return nil, nil
}

// Get all the processes of an app
func (s *appServer) GetProcesses(ctx context.Context, in *mPb.AppIndex) (*mPb.AppProcs, error) {
	if _, ok := s.mng.apps[in.Id]; !ok {
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.Id))
	}

	procs := make(map[string]*mPb.ProcList)
	for _, ag := range s.mng.agents {
		// Create a grpc client to an agent
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(fmt.Sprintf(
			"%s:%d", ag.Ip, ag.Port), opts...)
		if err != nil {
			grpclog.Printf("Failed to dial agent: %v", err)
		}
		defer conn.Close()
		client := agPb.NewMonitorProcsClient(conn)

		// Get the app's processes on the agent
		grpclog.Printf("Collect processes of app %s on agent %s", in.Id, ag.Id)
		list, err := client.GetProcesses(context.Background(),
			&agPb.ProcRequest{AppId: in.Id})
		if err != nil {
			grpclog.Printf("%v.GetProcesses(_) = _, %v", client, err)
			continue
		}

		procs[ag.Id] = &mPb.ProcList{Procs: make([]*mPb.Process, len(list.Procs))}
		for i, p := range list.Procs {
			procs[ag.Id].Procs[i] = &mPb.Process{
				Pid:  p.Pid,
				Name: p.Name,
				Cmd:  p.Cmd,
			}
		}
	}

	return &mPb.AppProcs{NodeProcs: procs}, nil
}
