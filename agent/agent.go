package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/olivere/elastic.v3"

	"github.com/lizhongz/dmoni/common"
	"github.com/lizhongz/dmoni/detector"
	pb "github.com/lizhongz/dmoni/proto/manager"
)

var (
	HbInterval   = time.Second * 6 // Heartbeat time interval
	MoniInterval = time.Second * 3 // Monitoring time interval
	// Local directory to store monitored data
	outDir = "/tmp/dmoni"
)

// Application info used for monitoring
type App struct {
	// Application Id
	Id string
	// Frameworks used by this application
	Frameworks []string
	// Job Ids in corresponding frameworks
	JobIds []string
	// Pid of the application's main process, if it's zero
	// means the process is not on this node.
	// TODO: use int instead of int32
	EntryPid int32
	// Processes of the app
	Procs []common.Process
	// Output file of monitored data
	ofile string

	// Executable name
	exe string
	// arguments for running the app
	args []string
}

type AppMap struct {
	m map[string]*App
	sync.RWMutex
}

type Agent struct {
	// monitored Applications
	apps *AppMap
	// Process detectors for different frameworks
	// (key, value) = (framework, detector)
	detectors map[string]detector.Detector
	// My node info
	me common.Node
	// Managero node info
	manager common.Node
	// Agent server
	server *agentServer

	//sync.RWMutex
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
		apps: &AppMap{m: make(map[string]*App)},
		detectors: map[string]detector.Detector{
			"hadoop": &detector.HadoopDetector{},
			"spark":  new(detector.SparkDetector),
		},
		manager: common.Node{Ip: cfg.MngIp, Port: cfg.MngPort},
		me: common.Node{
			Id: cfg.Id, Ip: cfg.Ip, Port: cfg.Port, Heartbeat: 0},
	}
	ag.server = newServer(ag)

	return ag
}

func (ag *Agent) Run() {

	// Check if data output direcotry exists. If not, create one
	err := os.Mkdir(outDir, 0774)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create output dir %s: %v", outDir, err)
	}

	// Start agent's server
	go ag.server.Run()

	// Connect to manager and maintain the conenction
	go ag.cast()

	// Start monitoring
	ag.Monitor()
}

func (ag *Agent) launch(appId string, exe string, arg ...string) error {
	// Run application as a child process
	cmd := exec.Command(exe, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to execute %s %v: %v", exe, arg, err)
		return err
	}

	app := &App{
		Id:       appId,
		exe:      exe,
		args:     arg,
		EntryPid: int32(cmd.Process.Pid),
	}
	ag.apps.Lock()
	//ag.apps.m[app.Id] = app
	ag.apps.Unlock()

	go func() {
		err = cmd.Wait()
		if err != nil {
			log.Printf("App %s exits with error: %v", app.Id, err)
		} else {
			log.Printf("App %s exits", app.Id)
		}

		// TODO: Notify manager

		ag.apps.Lock()
		//delete(ag.apps.m, app.Id)
		ag.apps.Unlock()
	}()

	return nil
}

// Monitoring all applications
func (ag *Agent) Monitor() {
	for {
		for _, app := range ag.apps.m {
			log.Printf("Monitoring app %s", app.Id)
			procs := make([]common.Process, 0)
			if app.EntryPid != 0 {
				procs = append(procs, common.Process{Pid: app.EntryPid})
			}

			//TODO(lizhong): using golang's io Process to monitor the main process

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
			app.Procs = procs

			f, err := os.OpenFile(app.ofile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
			if err != nil {
				log.Printf("Failed to create output file of app %s: %v", app.Id, err)
			}
			var buf bytes.Buffer

			// Monitor the app's processes
			for _, p := range app.Procs {
				//TODO(lizhong): configure the path of monitor.py
				cmd := exec.Command("python",
					"/home/lnz5/workspace/snapshot/app/monitor.py",
					"-n", "1", strconv.Itoa(int(p.Pid)))
				cmd.Stdout = &buf
				if err := cmd.Run(); err != nil {
					log.Print("Failed to get process snapshot: %v", err)
					if app.EntryPid != 0 && p.Pid == app.EntryPid {
						// Notify The application is finished or the entry
						// process does not exist.
						go ag.notifyDone(app.Id)
						break
					}
					continue
				}

				// Store process's performance data in a local file
				var data interface{}
				err = json.Unmarshal(buf.Bytes(), &data)
				if err != nil {
					log.Printf("Failed to unmarshal process snapshot %s: %v", buf.Bytes(), err)
					continue
				}
				m := data.(map[string]interface{})
				m["node"] = ag.me.Ip
				m["app_id"] = app.Id
				buf.Reset()

				out, err := json.Marshal(m)
				if err != nil {
					log.Printf("Failed to marshal %v: %v", m, err)
					continue
				}
				_, err = f.Write(out)
				if err != nil {
					log.Printf("Failed to write file %s: %v", app.ofile, err)
					continue
				}
			}
			f.Close()
		}

		time.Sleep(MoniInterval)
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

// storeData reads monitored data of an app, and push them to central database.
func (ag *Agent) storeData(app *App) error {
	// Open the file storing monitored data
	f, err := os.Open(app.ofile)
	defer f.Close()
	if err != nil {
		log.Printf("Failed to open file %s: %v", app.ofile, err)
		return err
	}

	// Create an ElasticSearch client
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL("http://129.6.57.225:9200"))
	if err != nil {
		log.Printf("Failed to create ElasticSearch client: %v", err)
		return err
	}

	// Check the existence of index; if not create one
	exist, err := client.IndexExists("dmoni").Do()
	if err != nil {
		log.Fatalf("Failed to call IndexExists: %v", err)
		return err
	}
	if !exist {
		_, err = client.CreateIndex("dmoni").Do()
		if err != nil {
			log.Fatalf("Failed to create index: %v", err)
			return err
		}
	}

	// Read, decode and store JSON stream from the file
	dec := json.NewDecoder(f)
	var m map[string]interface{}
	for dec.More() {
		if err := dec.Decode(&m); err != nil {
			log.Printf("Failed to decode: %v", err)
			return err
		}

		_, err = client.Index().
			Index("dmoni").Type("proc").
			BodyJson(m).Refresh(true).Do()
		if err != nil {
			log.Printf("Failed to store proc doc: %v", err)
			return err
		}
	}
	return nil
}
