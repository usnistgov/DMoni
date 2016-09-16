package agent

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	"github.com/lizhongz/dmoni/detector"
	pb "github.com/lizhongz/dmoni/proto/manager"
)

var (
	HbInterval = time.Second * 6 // Heartbeat time interval
)

type Agent struct {
	// List of monitored applications
	apps map[string]*common.App
	// Process detectors for different frameworks
	// (key, value) = (framework, detector)
	detectors map[string]detector.Detector
	// Application processes
	// (key, value) = (app id, process list)
	appProcs map[string][]common.Process

	// My node info
	me common.Node
	// Managero node info
	manager common.Node

	// Agent server
	server *agentServer

	sync.RWMutex
}

type Config struct {
	Id   string
	Ip   string
	Port int32

	// Manager's ip and port
	MngIp   string
	MngPort int32
}

// Create an agent given manager node's address (IP and port)
func NewAgent(cfg *Config) *Agent {
	ag := &Agent{
		apps: make(map[string]*common.App),
		detectors: map[string]detector.Detector{
			"hadoop": &detector.HadoopDetector{},
			"spark":  new(detector.SparkDetector),
		},
		appProcs: make(map[string][]common.Process),
		manager:  common.Node{Ip: cfg.MngIp, Port: cfg.MngPort},
		me: common.Node{
			Id: cfg.Id, Ip: cfg.Ip, Port: cfg.Port, Heartbeat: 0},
	}
	ag.server = newServer(ag)

	return ag
}

func (ag *Agent) Run() {
	// Start agent's server
	go ag.server.Run()

	// Connect to manager and maintain the conenction
	go ag.cast()

	// Start monitoring
	ag.Monitor()
}

// Monitoring all applications
func (ag *Agent) Monitor() {
	for {
		for _, app := range ag.apps {
			log.Printf("Monitoring app %s", app.Id)
			procs := make([]common.Process, 0)
			if app.EntryPid != 0 {
				procs = append(procs, common.Process{Pid: app.EntryPid})
			}

			// Detect all the running processes for each framework of this app
			for i, fw := range app.Frameworks {
				fps, err := ag.detectors[fw].Detect(app.JobIds[i])
				if err != nil {
					log.Printf("Failed to detect application' %s processes. Error: %s",
						app.Id, err)
					continue
				}
				procs = append(procs, fps...)
			}
			ag.appProcs[app.Id] = procs

			// Monitor the app's processes
			for _, p := range ag.appProcs[app.Id] {
				//TODO(lizhong): configure the path of monitor.py
				cmd := exec.Command("python",
					"/home/lnz5/workspace/snapshot/app/monitor.py",
					"-n", "1", strconv.Itoa(int(p.Pid)))
				var out bytes.Buffer
				cmd.Stdout = &out
				err := cmd.Run()
				if err != nil {
					log.Print("Failed to get process snapshot: %v", err)
					if app.EntryPid != 0 && p.Pid == app.EntryPid {
						// Notify The application is finished or the entry
						// process does not exist.
						ag.notifyDone(app.Id)

					}
					continue
				}
				//TODO(lizhong): Storing process info
				//log.Printf("Monitoring: %s %d %s %s\n",
				//  app.Id, p.Pid, p.ShortName, p.FullName)
			}
		}

		//TODO(lizhong): make the interval configable
		time.Sleep(3 * time.Second)
	}
}

// Cast sends agent's information including heartbeat value
// to manager periodly, in order to let manager know he is
// alive.
func (ag *Agent) cast() {
	// Create a grpc connection
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf(
		"%s:%d", ag.manager.Ip, ag.manager.Port), opts...)
	if err != nil {
		// If failed to dial, retry later
		grpclog.Printf("Failed to dial manager: %v", err)
		time.Sleep(HbInterval)
		go ag.cast()
		return
	}
	defer conn.Close()
	client := pb.NewManagerClient(conn)

	for {
		// Say Hi to manager
		grpclog.Printf("Say Hi to manager %s:%d",
			ag.manager.Ip, ag.manager.Port)
		ma, err := client.SayHi(
			context.Background(),
			&pb.NodeInfo{
				Id:        ag.me.Id,
				Ip:        ag.me.Ip,
				Port:      ag.me.Port,
				Heartbeat: ag.me.Heartbeat,
			})
		if err != nil {
			grpclog.Printf("%v.SayHi(_) = _, %v: ", client, err)
			time.Sleep(HbInterval)
			go ag.cast()
			return
		}

		// Update manager's info
		ag.manager.Id = ma.Id
		ag.manager.Heartbeat = ma.Heartbeat

		time.Sleep(HbInterval)
	}
}

// nitifyDone send an app's Id to mananger and tell him that
// an applicaiton is finished.
func (ag *Agent) notifyDone(appId string) {
	// Create a grpc connection
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf(
		"%s:%d", ag.manager.Ip, ag.manager.Port), opts...)
	if err != nil {
		grpclog.Printf("Failed to dial manager: %v", err)
		return
	}
	defer conn.Close()
	client := pb.NewManagerClient(conn)

	//TODO(lizhong): if failed to notify mananger, it should be
	//able to retry later.

	log.Printf("Notify mananger app %s is done", appId)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	_, err = client.NotifyDone(ctx, &pb.AppIndex{Id: appId})
	if err != nil {
		grpclog.Printf("Failed to notify mananger that app %s was done", appId)
		return
	}
}
