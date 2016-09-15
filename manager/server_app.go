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
	grpclog.Printf("Registering app %s", id)

	// Register the the application on manager
	app := &common.App{
		Id:         id,
		Frameworks: make([]string, len(in.Frameworks)),
	}
	copy(app.Frameworks, in.Frameworks)
	s.mng.apps.Lock()
	s.mng.apps.m[id] = app
	s.mng.apps.Unlock()

	// Register the app with all agents
	s.mng.agents.RLock()
	var wg sync.WaitGroup
	wg.Add(len(s.mng.agents.m))
	for _, ag := range s.mng.agents.m {
		go func(ag *common.Node) {
			defer wg.Done()
			// Create an agent client
			client, closeConn, err := getAgentClient(ag.Ip, ag.Port)
			if err != nil {
				log.Printf("Failed getAgentClient(): %v", err)
				return
			}
			defer closeConn()

			// Send app info to agent
			ai := &agPb.AppInfo{
				Id:         app.Id,
				Frameworks: make([]string, len(app.Frameworks)),
			}
			copy(ai.Frameworks, app.Frameworks)

			//grpclog.Printf("Register app %s with agent %s:%d", app.Id, ag.Id, ag.Port)
			newCtx, cancel := context.WithTimeout(ctx, time.Second*1)
			defer cancel()
			_, err = client.Register(newCtx, ai)
			if err != nil {
				grpclog.Printf("%v.Register(_) = _, %v", client, err)
				return
			}
		}(ag)
	}
	s.mng.agents.RUnlock()
	wg.Wait()

	// TODO(lizhong): if failed to propagate app info to agents
	return &mPb.AppIndex{Id: id}, nil
}

// Deregister an app
func (s *appServer) Deregister(ctx context.Context, in *mPb.AppIndex) (*mPb.DeregReply, error) {
	s.mng.apps.RLock()
	if _, ok := s.mng.apps.m[in.Id]; !ok {
		s.mng.apps.RUnlock()
		return nil, errors.New(fmt.Sprintf("App %s does not exist", in.Id))
	}
	s.mng.apps.RUnlock()
	log.Printf("Deregister app %s", in.Id)

	// Deregister the app with all agents
	s.mng.agents.RLock()
	var wg sync.WaitGroup
	wg.Add(len(s.mng.agents.m))
	for _, ag := range s.mng.agents.m {
		// Create a grpc client to an agent
		go func(ag *common.Node) {
			defer wg.Done()
			// Create an agent client
			client, closeConn, err := getAgentClient(ag.Ip, ag.Port)
			if err != nil {
				log.Printf("Failed getAgentClient(): %v", err)
				return
			}
			defer closeConn()

			// Degregister the app on the agent
			//grpclog.Printf("Deregister app %s on with agent %s", in.Id, ag.Id)
			newCtx, cancel := context.WithTimeout(ctx, time.Second*1)
			defer cancel()
			_, err = client.Deregister(newCtx, &agPb.DeregRequest{AppId: in.Id})
			if err != nil {
				grpclog.Printf("%v.Deregister(_) = _, %v", client, err)
				return
			}
		}(ag)
	}
	s.mng.agents.RUnlock()
	wg.Wait()

	s.mng.apps.Lock()
	delete(s.mng.apps.m, in.Id)
	s.mng.apps.Unlock()
	return &mPb.DeregReply{}, nil
}

// Get the running status of an app
func (s *appServer) GetStatus(ctx context.Context, in *mPb.AppIndex) (*mPb.AppStatus, error) {
	return nil, nil
}

// Get all the processes of an app
func (s *appServer) GetProcesses(ctx context.Context, in *mPb.AppIndex) (*mPb.AppProcs, error) {
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
	procs := make(map[string]*mPb.ProcList)
	for ag := range agCh {
		procs[ag.id] = &mPb.ProcList{Procs: make([]*mPb.Process, len(ag.reply.Procs))}
		//log.Printf("processes from agent %s", ag.id)
		for i, p := range ag.reply.Procs {
			procs[ag.id].Procs[i] = &mPb.Process{
				Pid:  p.Pid,
				Name: p.Name,
				Cmd:  p.Cmd,
			}
		}
	}
	return &mPb.AppProcs{NodeProcs: procs}, nil
}

// Test used to verify if server is blocked or not
func (s *appServer) Test(ctx context.Context, in *mPb.TRequest) (*mPb.TReply, error) {
	grpclog.Println("GRPC Test()")
	return &mPb.TReply{}, nil
}

// getAgentClient returns a given agent's client
func getAgentClient(ip string, port int32) (agPb.MonitorProcsClient, func() error, error) {
	//TODO(lizhong): Reuse connections to agents

	// Create a grpc client to an agent
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", ip, port), opts...)
	if err != nil {
		//grpclog.Printf("Failed to dial agent: %v", err)
		return nil, nil, err
	}
	return agPb.NewMonitorProcsClient(conn), func() error { return conn.Close() }, nil
}
