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
	"gopkg.in/olivere/elastic.v3"

	"github.com/lizhongz/dmoni/common"
	agPb "github.com/lizhongz/dmoni/proto/agent"
)

// Application info used for monitoring
type App struct {
	// Application Id
	Id string
	// Frameworks used by this application
	Frameworks []string
	// Job Ids in corresponding frameworks
	JobIds []string
	// IP address of the node where the app started
	EntryNode string
	// Pid of the application's main process
	EntryPid int
	// Status of the application: running, exited
	Status string

	// Flag indicating if monitoring the application is enbaled
	monitored bool
	// Flag indicating if the app is launched by dmoni
	launched bool

	// Executable name
	ExecName string
	// Execution arguments
	ExecArgs []string
}

type appMap struct {
	m map[string]*App
	sync.RWMutex
}

func newAppMap() *appMap {
	return &appMap{m: make(map[string]*App)}
}

type agentMap struct {
	m map[string]*common.Node
	sync.RWMutex
}

func newAgentMap() *agentMap {
	return &agentMap{m: make(map[string]*common.Node)}
}

type Manager struct {
	// Submitted or registered applications
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

// monitor call Register RPC of agents to start to monitor the
// app's processes.
func (m *Manager) monitor(ctx context.Context, app *App) error {
	// reg registers the app with a agent
	reg := func(ag *common.Node) error {
		// Create an agent client
		client, closeConn, err := getAgentClient(ag.Ip, ag.Port)
		if err != nil {
			log.Printf("Failed getAgentClient(): %v", err)
			return err
		}
		defer closeConn()

		// Send app info to agent
		ai := &agPb.AppInfo{
			Id:         app.Id,
			Frameworks: app.Frameworks,
			JobIds:     app.JobIds,
		}
		_, err = client.Register(ctx, ai)
		if err != nil {
			grpclog.Printf("%v.Register(_) = _, %v", client, err)
			return err
		}
		return nil
	}

	// Register the app with all agents
	m.agents.RLock()
	var wg sync.WaitGroup
	wg.Add(len(m.agents.m))
	for _, ag := range m.agents.m {
		go func(ag *common.Node) {
			defer wg.Done()
			if err := reg(ag); err != nil {
				log.Printf("Failed to register app %s with (%s, %s): %v",
					app.Id, ag.Id, ag.Ip, err)
			}
		}(ag)
	}
	m.agents.RUnlock()
	wg.Wait()

	// TODO: handle failures of registration with agents

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

func (m *Manager) logApp(data map[string]interface{}) error {
	// Create an ElasticSearch client
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL("http://192.168.0.3:9200"))
	if err != nil {
		log.Printf("Failed to create ElasticSearch client: %v", err)
		return err
	}

	_, err = client.Index().
		Index("dmoni").Type("app").
		BodyJson(data).Refresh(true).Do()
	if err != nil {
		log.Printf("Failed to store data in ElasticSearch: %v", err)
		return err
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
