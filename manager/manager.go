package manager

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	agPb "github.com/lizhongz/dmoni/proto/agent"
)

type Manager struct {
	// The server providing services for client and agents
	server *managerServer
	// Monitored applications
	apps map[string]*common.App

	// Cluster nodes info
	me     *common.Node
	agents map[string]*common.Node
}

type Config struct {
	Id   string
	Ip   string
	Port int32
}

func NewManager(cfg *Config) *Manager {
	m := new(Manager)
	m.apps = make(map[string]*common.App)
	m.me = &common.Node{
		Id:        cfg.Id,
		Ip:        cfg.Ip,
		Port:      cfg.Port,
		Heartbeat: 0,
		Timestamp: time.Now(),
	}
	m.server = newServer(m)
	m.agents = make(map[string]*common.Node)
	return m
}

func (m *Manager) Run() {

	go m.server.Run()

	// TODO(lizhong): check if agents are alive

	// Testing: Register an app

	app := &common.App{
		Id:         "we3923xc92",
		Frameworks: []string{"spark"},
	}

	time.Sleep(3 * time.Second)
	m.register(app)
	time.Sleep(5 * time.Second)

	for i := 0; i < 10; i++ {
		_, err := m.collectAppProcs(app.Id)
		if err != nil {
			grpclog.Printf("Failed to collect app %s's processes: %v", err)
		}
		time.Sleep(3 * time.Second)
	}

	m.deregister(app.Id)

	for {
	}
}

func (m *Manager) register(app *common.App) error {
	if _, ok := m.apps[app.Id]; ok {
		return errors.New(fmt.Sprintf("Id %d has already registered", app.Id))
	}

	m.apps[app.Id] = app

	// Register the app with all agents
	for _, ag := range m.agents {
		// Create a grpc client
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(fmt.Sprintf(
			"%s:%d", ag.Ip, ag.Port), opts...)
		if err != nil {
			grpclog.Printf("Failed to dial agent: %v", err)
		}
		defer conn.Close()
		client := agPb.NewMonitorProcsClient(conn)

		// Send app info

		ai := &agPb.AppInfo{
			Id:         app.Id,
			Frameworks: make([]string, len(app.Frameworks)),
		}
		copy(ai.Frameworks, app.Frameworks)

		grpclog.Printf("Register app %s on with agent %s", app.Id, ag.Id)
		_, err = client.Register(context.Background(), ai)
		if err != nil {
			grpclog.Printf("%v.Register(_) = _, %v", client, err)
			continue
		}
	}

	return nil
}

func (m *Manager) deregister(appId string) error {
	// Deregister the app with all agents
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
		grpclog.Printf("Deregister app %s on with agent %s", appId, ag.Id)
		_, err = client.Deregister(context.Background(),
			&agPb.DeregRequest{AppId: appId})
		if err != nil {
			grpclog.Printf("%v.Deregister(_) = _, %v", client, err)
			continue
		}
	}
	return nil
}

func (m *Manager) collectAppProcs(appId string) ([]common.Process, error) {
	procs := make([]common.Process, 0)
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

		// Get the app's processes on the agent
		grpclog.Printf("Collect processes of app %s on agent %s", appId, ag.Id)
		list, err := client.GetProcesses(context.Background(),
			&agPb.ProcRequest{AppId: appId})
		if err != nil {
			grpclog.Printf("%v.GetProcesses(_) = _, %v", client, err)
			continue
		}

		for _, p := range list.Procs {
			grpclog.Printf("agent %s process (%d %s %s)", ag.Id, p.Pid, p.Name, p.Cmd)
			procs = append(procs, common.Process{
				Pid:       p.Pid,
				ShortName: p.Name,
				FullName:  p.Cmd,
			})
		}
	}

	return procs, nil
}
