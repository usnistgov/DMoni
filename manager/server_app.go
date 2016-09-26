package manager

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/satori/go.uuid"
	//"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	agPb "github.com/lizhongz/dmoni/proto/agent"
	appPb "github.com/lizhongz/dmoni/proto/app"
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
	appPb.RegisterAppGaugeServer(grpcServer, s)
	grpcServer.Serve(lis)
}

// Submit an application
func (s *appServer) Submit(ctx context.Context, in *appPb.SubRequest) (*appPb.AppIndex, error) {
	// Generate an id for the application
	id := strings.Join(strings.Split(uuid.NewV4().String(), "-"), "")
	grpclog.Printf("Launching app %s", id)

	app := &App{
		Id:        id,
		ExecName:  in.ExecName,
		ExecArgs:  in.ExecArgs,
		monitored: in.Moni,
	}

	node := s.mng.findNode(in.Ip)
	if node == nil {
		return nil, errors.New(fmt.Sprintf("Node %d does not exist", in.Ip))
	}
	client, close, err := getAgentClient(node.Ip, node.Port)
	if err != nil {
		log.Printf("Failed to get agent %s's client: %v", node.Id, err)
		return nil, errors.New(fmt.Sprintf("Failed to connect to %s", in.Ip))
	}
	defer close()

	_, err = client.Launch(ctx,
		&agPb.LchRequest{
			AppId:    id,
			ExecName: in.ExecName,
			ExecArgs: in.ExecArgs,
			Moni:     in.Moni,
		})
	if err != nil {
		log.Printf("Failed to launch app: %v", err)
		return nil, errors.New(fmt.Sprintf("Failed to launch app: %v", err))
	}

	s.mng.apps.Lock()
	s.mng.apps.m[id] = app
	s.mng.apps.Unlock()

	if in.Moni {
		// Register app on all node for monitoring its processes
	}

	return &appPb.AppIndex{Id: id}, nil
}

// Kill an application
func (s *appServer) Kill(ctx context.Context, in *appPb.AppIndex) (*appPb.KillReply, error) {
	return nil, nil
}

// Register an application
func (s *appServer) Register(ctx context.Context, in *appPb.AppDesc) (*appPb.AppIndex, error) {

	// If the enrty node ip address is not empty string in AppDesc,
	// then check the existence of the given node.
	// Otherwise, assume user want to monitoring only the framework,
	// or a job of the framework.
	var entry *common.Node
	if strings.Compare(in.EntryNode, "") != 0 {
		entry = s.mng.findNode(in.EntryNode)
		if entry == nil {
			return nil, errors.New(fmt.Sprintf(
				"Entry node %s does not exist", in.EntryNode))
		}
	}

	// Generate an id for the application
	id := strings.Join(strings.Split(uuid.NewV4().String(), "-"), "")
	grpclog.Printf("Registering app %s", id)

	app := &App{
		Id:         id,
		Frameworks: in.Frameworks,
		JobIds:     in.JobIds,
		EntryNode:  in.EntryNode,
		EntryPid:   int(in.EntryPid),
	}

	reg := func(ag *common.Node, pid int) error {
		// Create an agent client
		client, closeConn, err := getAgentClient(ag.Ip, ag.Port)
		if err != nil {
			log.Printf("Failed getAgentClient(): %v", err)
			return err
		}
		defer closeConn()

		//grpclog.Printf("Register app %s with agent %s:%d", app.Id, ag.Id, ag.Port)

		// Send app info to agent
		ai := &agPb.AppInfo{
			Id:         app.Id,
			Frameworks: app.Frameworks,
			JobIds:     app.JobIds,
			Pid:        int64(pid),
		}
		_, err = client.Register(ctx, ai)
		if err != nil {
			grpclog.Printf("%v.Register(_) = _, %v", client, err)
			return err
		}
		return nil
	}

	// Register the app with all agents
	s.mng.agents.RLock()
	var wg sync.WaitGroup
	wg.Add(len(s.mng.agents.m))
	for _, ag := range s.mng.agents.m {
		go func(ag *common.Node) {
			defer wg.Done()
			pid := 0
			if entry != nil && strings.Compare(ag.Id, entry.Id) == 0 {
				pid = int(in.EntryPid)
			}
			// TODO(lizhong): if failed to propagate app info to agents
			_ = reg(ag, pid)
		}(ag)
	}
	s.mng.agents.RUnlock()
	wg.Wait()

	// Register the app on manager
	s.mng.apps.Lock()
	s.mng.apps.m[id] = app
	s.mng.apps.Unlock()

	return &appPb.AppIndex{Id: id}, nil
}

// Deregister an app
func (s *appServer) Deregister(ctx context.Context, in *appPb.AppIndex) (*appPb.DeregReply, error) {
	err := s.mng.deregister(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	return &appPb.DeregReply{}, nil
}

// Get the running status of an app
func (s *appServer) GetStatus(ctx context.Context, in *appPb.AppIndex) (*appPb.AppStatus, error) {
	return nil, nil
}

// Get all the processes of an app
func (s *appServer) GetProcesses(ctx context.Context, in *appPb.AppIndex) (*appPb.AppProcs, error) {
	s.mng.apps.RLock()
	if _, ok := s.mng.apps.m[in.Id]; !ok {
		s.mng.apps.RUnlock()
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.Id))
	}
	s.mng.apps.RUnlock()

	type agProcs struct {
		id    string
		reply *agPb.ProcList
	}
	agCh := make(chan *agProcs)

	s.mng.agents.RLock()
	var wg sync.WaitGroup
	wg.Add(len(s.mng.agents.m))
	for _, ag := range s.mng.agents.m {
		// Send requests to each agent
		go func(ag *common.Node) {
			defer wg.Done()
			// Create an agent client
			client, closeConn, err := getAgentClient(ag.Ip, ag.Port)
			if err != nil {
				log.Printf("Failed getAgentClient(): %v", err)
				return
			}
			defer closeConn()

			// Get the app's processes on the agent
			//grpclog.Printf("Collect processes of app %s on agent %s", in.Id, ag.Id)
			newCtx, cancel := context.WithTimeout(ctx, time.Second*1)
			defer cancel()
			list, err := client.GetProcesses(newCtx, &agPb.ProcRequest{AppId: in.Id})
			if err != nil {
				grpclog.Printf("%v.GetProcesses(_) = _, %v", client, err)
				return
			}
			agCh <- &agProcs{id: ag.Id, reply: list}
		}(ag)
	}
	s.mng.agents.RUnlock()

	go func() {
		wg.Wait()
		close(agCh)
	}()

	// Retrieve the resulst from previous requests
	procs := make(map[string]*appPb.ProcList)
	for ag := range agCh {
		procs[ag.id] = &appPb.ProcList{Procs: make([]*appPb.Process, len(ag.reply.Procs))}
		//log.Printf("processes from agent %s", ag.id)
		for i, p := range ag.reply.Procs {
			procs[ag.id].Procs[i] = &appPb.Process{
				Pid:  p.Pid,
				Name: p.Name,
				Cmd:  p.Cmd,
			}
		}
	}
	return &appPb.AppProcs{NodeProcs: procs}, nil
}

// Test used to verify if server is blocked or not
func (s *appServer) Test(ctx context.Context, in *appPb.TRequest) (*appPb.TReply, error) {
	grpclog.Println("GRPC Test()")
	return &appPb.TReply{}, nil
}
