package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/olivere/elastic.v3"

	"github.com/lizhongz/dmoni/common"
	"github.com/lizhongz/dmoni/detector"
	"github.com/lizhongz/dmoni/manager"
	pb "github.com/lizhongz/dmoni/proto/manager"
)

var (
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
	EntryPid int
	// Processes of the app
	Procs []common.Process
	// Output file of monitored data
	ofile string
	// Context used to check if the application is killed on purpose
	ctx context.Context
	// function used to stop/cancel the application
	cancel func()

	// Executable name
	exe string
	// arguments for running the app
	args []string
	// Start time
	stime time.Time
	// End time
	etime time.Time
	// stdout of app's main process
	sout *bytes.Buffer
	// stderr of app's main process
	serr *bytes.Buffer
}

type AppMap struct {
	m map[string]*App
	sync.RWMutex
}

type Agent struct {
	// Launched applications by agent
	lchApps *AppMap
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
	// Data storage server's address
	dsAddr string

	//sync.RWMutex
}

type Config struct {
	Id   string
	Ip   string
	Port int32

	// Manager's ip and port
	MngIp   string
	MngPort int32

	// Data storage's address
	DsAddr string
}

// Create an agent given manager node's address (IP and port)
func NewAgent(cfg *Config) *Agent {
	ag := &Agent{
		apps:    &AppMap{m: make(map[string]*App)},
		lchApps: &AppMap{m: make(map[string]*App)},
		detectors: map[string]detector.Detector{
			"hadoop": &detector.HadoopDetector{},
			"spark":  new(detector.SparkDetector),
		},
		manager: common.Node{Ip: cfg.MngIp, Port: cfg.MngPort},
		me: common.Node{
			Id: cfg.Id, Ip: cfg.Ip, Port: cfg.Port, Heartbeat: 0},
		dsAddr: cfg.DsAddr,
	}
	ag.server = newServer(ag)

	return ag
}

func (ag *Agent) Run() {
	log.Printf("Dmoni Agent")

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

// Monitoring all applications
//
// Monitroing an applcation at a time can be divided in to three stages:
// 1) detect the application's processes;
// 2) measure each process's resource usage;
// 3) log the measured info.
func (ag *Agent) Monitor() {
	snap := func(app *App) {
		// Detect application's processes
		procs := make([]common.Process, 0)
		if a, present := ag.lchApps.m[app.Id]; present {
			// Include the entry process of the app
			procs = append(procs, common.Process{
				Pid:       a.EntryPid,
				ShortName: a.exe,
				FullName:  fmt.Sprintf("%s %s", a.exe, strings.Join(a.args, " ")),
			})
		}
		for i, fw := range app.Frameworks {
			// Detect pocesses of this framework
			fps, err := ag.detectors[fw].Detect(app.JobIds[i])
			if err != nil {
				log.Printf("Failed to detect application' %s processes. Error: %s",
					app.Id, err)
				continue
			}
			procs = append(procs, fps...)
		}
		app.Procs = procs

		// Create channel for measuring process info
		dataCh := make(chan *bytes.Buffer, 1)
		defer close(dataCh)
		// Start a dataLoggoer to store process info
		go ag.dataLogger(app.Id, app.ofile, dataCh)

		// measure each process's resource usage
		for _, p := range app.Procs {
			var buf bytes.Buffer
			//TODO(lizhong): configure the path of monitor.py
			cmd := exec.Command("python",
				"/home/lnz5/workspace/snapshot/app/monitor.py",
				"-n", "1", strconv.Itoa(int(p.Pid)))
			cmd.Stdout = &buf
			if err := cmd.Run(); err != nil {
				log.Printf("Failed to get process snapshot: %v", err)
				continue
			}
			// push measured info to output channel
			dataCh <- &buf
		}
	}

	for {
		ag.apps.RLock()
		// For each application
		for _, app := range ag.apps.m {
			log.Printf("Monitoring app %s", app.Id)
			go snap(app)
		}
		ag.apps.RUnlock()

		time.Sleep(manager.MoniInterval)
	}
}

// launch runs an application as a subprocess.
func (ag *Agent) launch(appId string, exe string, arg ...string) (err error) {
	app := &App{
		Id:   appId,
		exe:  exe,
		args: arg,
		sout: bytes.NewBuffer(make([]byte, 0, 1024)),
		serr: bytes.NewBuffer(make([]byte, 0, 1024)),
	}

	// Run application as a child process
	app.ctx, app.cancel = context.WithCancel(context.Background())
	cmd := exec.CommandContext(app.ctx, exe, arg...)
	cmd.Stdout = app.sout
	cmd.Stderr = app.serr
	err = cmd.Start()
	if err != nil {
		log.Printf("Failed to execute %s %v: %v", exe, arg, err)
		return err
	}
	app.stime = time.Now()
	app.EntryPid = cmd.Process.Pid

	ag.lchApps.Lock()
	ag.lchApps.m[app.Id] = app
	ag.lchApps.Unlock()

	go func() {
		// Wait the application exits
		err = cmd.Wait()
		if err != nil {
			log.Printf("App %s exits with error: %v", app.Id, err)
		} else {
			log.Printf("App %s exits", app.Id)
		}

		select {
		case <-app.ctx.Done():
			// if application is killed by manager
			log.Printf("App %s was killed", app.Id)
			return
		default:
			app.etime = time.Now()
			go ag.logApp(app)
			ag.notifyDone(app)

			ag.lchApps.Lock()
			delete(ag.lchApps.m, app.Id)
			ag.lchApps.Unlock()
		}
	}()

	return nil
}

// kill a launched application
func (ag *Agent) kill(app *App) {
	// Remove the app from launched list
	ag.lchApps.Lock()
	delete(ag.lchApps.m, app.Id)
	ag.lchApps.Unlock()

	// Stop the app's main process
	app.cancel()
}

// cast sends agent's information including heartbeat value
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
		time.Sleep(manager.HbInterval)
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
			time.Sleep(manager.HbInterval)
			go ag.cast()
			return
		}

		// Update manager's info
		ag.manager.Id = ma.Id
		ag.manager.Heartbeat = ma.Heartbeat

		time.Sleep(manager.HbInterval)
	}
}

// nitifyDone sends an app's Id to mananger and tell him that
// an applicaiton is finished.
func (ag *Agent) notifyDone(app *App) {
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

	log.Printf("Notify mananger app %s is done", app.Id)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	_, err = client.NotifyDone(ctx, &pb.NDRequest{
		AppId:     app.Id,
		StartTime: &timestamp.Timestamp{Seconds: app.stime.Unix()},
		EndTime:   &timestamp.Timestamp{Seconds: app.etime.Unix()},
	})
	if err != nil {
		grpclog.Printf("Failed to notify mananger that app %s was done: %v", app.Id, err)
		return
	}
}

// logApp stores the app's information in database.
func (ag *Agent) logApp(app *App) error {
	// Create an ElasticSearch client
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL(ag.dsAddr))
	if err != nil {
		log.Printf("Failed to create ElasticSearch client: %v", err)
		return err
	}

	st := app.stime.Format(time.RFC3339)
	et := app.etime.Format(time.RFC3339)
	data := map[string]interface{}{
		"app_id":     app.Id,
		"entry_node": ag.me.Ip,
		"exec":       app.exe,
		"args":       app.args,
		"start_at":   st,
		"end_at":     et,
		"stdout":     app.sout.String(),
		"stderr":     app.serr.String(),
		"timestamp":  time.Now().Format(time.RFC3339),
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

// dataLogger retrieves process info from a data channel and
// stores them in a temperary local file.
func (ag *Agent) dataLogger(appId, fname string, dataCh <-chan *bytes.Buffer) {
	// Create an temperary file
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to create output file %s: %v", fname, err)
	}
	defer f.Close()

	for buf := range dataCh {
		// Store process's performance data in a local file
		var data interface{}
		err = json.Unmarshal(buf.Bytes(), &data)
		if err != nil {
			log.Printf("Failed to unmarshal process snapshot %s: %v", buf.Bytes(), err)
			continue
		}
		m := data.(map[string]interface{})
		m["node"] = ag.me.Ip
		m["app_id"] = appId
		buf.Reset()

		out, err := json.Marshal(m)
		if err != nil {
			log.Printf("Failed to marshal %v: %v", m, err)
			continue
		}
		_, err = f.Write(out)
		if err != nil {
			log.Printf("Failed to write file %s: %v", fname, err)
			continue
		}
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
		elastic.SetURL(ag.dsAddr))
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

// getLchApp returns the pointer to a launched app
func (ag *Agent) getLchApp(id string) *App {
	ag.lchApps.RLock()
	defer ag.lchApps.RUnlock()
	return ag.lchApps.m[id]
}

// getMoniApp returns the pointer to a monitered app
func (ag *Agent) getMoniApp(id string) *App {
	ag.apps.RLock()
	defer ag.apps.RUnlock()
	return ag.apps.m[id]
}
