package agent

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	"github.com/lizhongz/dmoni/detector"
	pb "github.com/lizhongz/dmoni/proto/cluster"
)

type Agent struct {
	// List of monitored applications
	apps map[string]*common.App
	// Process detectors for different frameworks
	// (key, value) = (framework, detector)
	detectors map[string]detector.Detector
	// Application processes
	// (key, value) = (app id, process list)
	appProcs map[string][]detector.Process

	// My node info
	me common.Node
	// Managero node info
	manager common.Node
}

// Create an agent given manager node's address (IP and port)
func NewAgent(mIp string, mPort int32) *Agent {
	ag := &Agent{
		apps: make(map[string]*common.App),
		detectors: map[string]detector.Detector{
			"hadoop": &detector.HadoopDetector{},
			"spark":  new(detector.SparkDetector),
		},
		appProcs: make(map[string][]detector.Process),
		manager:  common.Node{Ip: mIp, Port: mPort},
		me:       common.Node{Id: "agent-0", Heartbeat: 0},
	}

	return ag
}

func (ag *Agent) Run() {
	// Say hi to manger periodically

	//go func() {
	// Create a membership client
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf(
		"%s:%d", ag.manager.Ip, ag.manager.Port), opts...)
	if err != nil {
		grpclog.Fatalf("Failed to dial manager: %v", err)
	}
	defer conn.Close()
	client := pb.NewMembershipClient(conn)

	for {
		grpclog.Printf("Say Hi to manager %s %s:%d",
			ag.manager.Id, ag.manager.Ip, ag.manager.Port)
		ma, err := client.SayHi(
			context.Background(),
			&pb.NodeInfo{
				Id:        ag.me.Id,
				Ip:        ag.me.Ip,
				Port:      ag.me.Port,
				Heartbeat: ag.me.Heartbeat,
			})
		if err != nil {
			grpclog.Fatalf("%v.SayHi(_) = _, %v: ", client, err)
		}

		time.Sleep(5 * time.Second)
	}
	//}()
}

// Register an application
func (ag *Agent) Register(app *common.App) {
	fws := make([]string, len(app.Frameworks))
	copy(fws, app.Frameworks)
	ag.apps[app.Id] = &common.App{
		Id:         app.Id,
		Frameworks: fws,
	}
}

// Unregister an application
func (ag *Agent) Unregister(appId string) {
	delete(ag.apps, appId)
}

// Monitoring all applications
func (ag *Agent) Monitor() {
	for {
		for _, app := range ag.apps {
			log.Print(app.Frameworks)
			// Detect all the running processes for each framework of this app
			for _, fw := range app.Frameworks {
				fps, err := ag.detectors[fw].Detect(app.Id)
				if err != nil {
					log.Printf("Failed to detect application' %s processes. Error: %s",
						app.Id, err)
				}
				ag.appProcs[app.Id] = fps
			}

			// Monitor the app's processes
			for _, p := range ag.appProcs[app.Id] {
				//TODO(lizhong): configure the path of monitor.py
				cmd := exec.Command("python",
					"/home/lnz5/workspace/snapshot/app/monitor.py",
					"-n", "1", p.Pid)
				var out bytes.Buffer
				cmd.Stdout = &out
				err := cmd.Run()
				if err != nil {
					log.Print(err)
				}
				log.Printf("%s %s %s %s\n",
					app.Id, p.Pid, p.ShortName, p.FullName)
				log.Print(out.String())
			}
		}

		time.Sleep(3 * time.Second)
	}
}
