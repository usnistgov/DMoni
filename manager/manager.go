package manager

import (
	"github.com/lizhongz/dmoni/common"
)

type Manager struct {
	// The server providing services for client and agents
	server *managerServer
	// Monitored applications
	apps map[string]*common.App
}

func NewManager() *Manager {
	m := new(Manager)
	m.server = newServer("manager", "192.168.0.6", 5300)
	m.apps = make(map[string]*common.App)
	return m
}

func (m *Manager) Run() {
	m.server.Run()
}
