package manager

import (
	"time"

	"github.com/lizhongz/dmoni/common"
)

type Manager struct {
	// Monitored applications
	apps map[string]*common.App

	// Cluster nodes info
	me     *common.Node
	agents map[string]*common.Node
	// The server providing services for agents
	masterServer *masterServer

	// The server providing services for client application
	appServer *appServer
	// Application server port
	appPort int32
}

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

	m.apps = make(map[string]*common.App)
	m.appPort = cfg.AppPort
	m.appServer = newAppServer(m)

	m.me = &common.Node{
		Id:        cfg.Id,
		Ip:        cfg.Ip,
		Port:      cfg.NodePort,
		Heartbeat: 0,
		Timestamp: time.Now(),
	}
	m.agents = make(map[string]*common.Node)
	m.masterServer = newMasterServer(m)

	return m
}

func (m *Manager) Run() {

	go m.masterServer.Run()

	go m.appServer.Run()

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

	for {
	}
}
