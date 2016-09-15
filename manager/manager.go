package manager

import (
	"sync"
	"time"

	"github.com/lizhongz/dmoni/common"
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
