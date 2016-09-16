package manager

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	agPb "github.com/lizhongz/dmoni/proto/agent"
)

type appMap struct {
	m map[string]*common.App
	sync.RWMutex
}

func newAppMap() *appMap {
	return &appMap{m: make(map[string]*common.App)}
}

type agentMap struct {
	m map[string]*common.Node
	sync.RWMutex
}

func newAgentMap() *agentMap {
	return &agentMap{m: make(map[string]*common.Node)}
}

type Manager struct {
	// Monitored applications
	apps *appMap

	// Cluster nodes info
	me     *common.Node
	agents *agentMap
	// The server providing services for agents
	masterServer *masterServer

	// The server providing services for client application
	appServer *appServer
	// Application server port
	appPort int32

	sync.RWMutex
}

// TODO(lizhong) Functionalities:
// - Register existing app on agents. Agents pull app list from manster?

type Config struct {
	// Manager's id
	Id string
	// Manager's IP address
	Ip string
	// Port for agent services
	NodePort int32
	// Port for app services
	AppPort int32
}

func NewManager(cfg *Config) *Manager {
	m := new(Manager)

	m.apps = newAppMap()
	m.appPort = cfg.AppPort
	m.appServer = newAppServer(m)

	m.me = &common.Node{
		Id:        cfg.Id,
		Ip:        cfg.Ip,
		Port:      cfg.NodePort,
		Heartbeat: 0,
		Timestamp: time.Now(),
	}
	m.agents = newAgentMap()
	m.masterServer = newMasterServer(m)

	return m
}

func (m *Manager) Run() {

	go m.masterServer.Run()

	m.appServer.Run()

	// TODO(lizhong): check if agents are alive

	// Testing: Register an app

	/*
		app := &common.App{
			Id:         "we3923xc92",
			Frameworks: []string{"spark", "hadoop"},
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
	*/
}

// deregister an application
func (m *Manager) deregister(ctx context.Context, appId string) error {
	m.apps.RLock()
	if _, ok := m.apps.m[appId]; !ok {
		m.apps.RUnlock()
		return errors.New(fmt.Sprintf("App %s does not exist", appId))
	}
	m.apps.RUnlock()
	log.Printf("Deregister app %s", appId)

	// Deregister the app with all agents
	m.agents.RLock()
	var wg sync.WaitGroup
	wg.Add(len(m.agents.m))
	for _, ag := range m.agents.m {
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
			_, err = client.Deregister(ctx, &agPb.DeregRequest{AppId: appId})
			if err != nil {
				grpclog.Printf("%v.Deregister(_) = _, %v", client, err)
				return
			}
		}(ag)
	}
	m.agents.RUnlock()
	wg.Wait()

	m.apps.Lock()
	delete(m.apps.m, appId)
	m.apps.Unlock()
	return nil
}

// findAgent returns node corresponding to a given ip address
func (m *Manager) findNode(ip string) *common.Node {
	m.agents.RLock()
	defer m.agents.RUnlock()
	for _, ag := range m.agents.m {
		if strings.Compare(ag.Ip, ip) == 0 {
			return ag
		}
	}
	return nil
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
