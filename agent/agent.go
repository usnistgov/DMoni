package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
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
	// Dmoni's path set from DMONIPATH env variable
	dmoniPath string
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
	// Application launch channel
	lch chan *AppExec

	// Heartbeat time interval in second
	hbItv time.Duration
	// Monitoring time Interval in second
	moniItv time.Duration

	//sync.RWMutex
}

type Config struct {
	// Agent's ID
	Id string
	// Agent's address
	Ip   string
	Port int32

	// Manager's ip and port
	MngIp   string
	MngPort int32

	// Data storage's address
	DsAddr string
}

type Doc struct {
	kind    string
	content interface{}
}

const (
	DocTypeProc = "proc"
	DocTypeSys  = "sys"
	DocTypeApp  = "app"
)

type AppExec struct {
	appId string
	cmd   string
	args  []string
	errCh chan error
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
		dsAddr:  cfg.DsAddr,
		hbItv:   manager.HbInterval,
		moniItv: manager.MoniInterval,
	}
	ag.server = newServer(ag)

	return ag
}

// checkConfig verifies the configurations
func (ag *Agent) checkConfig() {
	// Check if data output direcotry exists. If not, create one
	err := os.Mkdir(outDir, 0774)
	if err != nil && !os.IsExist(err) {
		log.Fatalf("Failed to create output dir %s: %v", outDir, err)
	}

	// Check if env variable DMONIPATH is defined
	dmoniPath = os.Getenv("DMONIPATH")
	if dmoniPath == "" {
		log.Fatalf("Environment variable DMONIPATH is empty")
	}
}

func (ag *Agent) Run() {
	ag.checkConfig()
	log.Printf("Dmoni Agent")

	// Create an app monitor to measure apps' resources usages
	procDocCh := ag.appMonitor()

	// Create a system monitor to measure system's resource usages
	sysDocCh := ag.sysMonitor()

	// Create a launcher to launch applications
	ag.lch = make(chan *AppExec, 1)
	ag.launch(ag.lch)

	// Merge collected info or doc from multiple sources
	docCh := ag.merger(procDocCh, sysDocCh)

	// Store collected docs to database
	ag.store(docCh)

	// Connect to manager and maintain the conenction
	go func() {
		time.Sleep(1 * time.Second)
		ag.cast()
	}()

	// Start agent's server
	ag.server.Run()
}

// appMonitor meausures all applications' resource usages
//
// Monitroing an applcation can be divided in to three stages:
// 1) detect the application's processes;
// 2) measure each process's resource usage;
// 3) log the measured info.
func (ag *Agent) appMonitor() <-chan *Doc {
	docCh := make(chan *Doc, 5)

	moniOne := func(app *App) {
		var buf bytes.Buffer
		app.Procs = ag.detectProcs(app)
		// measure each process's resource usage
		for _, p := range app.Procs {
			cmd := exec.Command("python", path.Join(dmoniPath,
				"snapshot/monitor.py"),
				"-n", "1", strconv.Itoa(int(p.Pid)))
			cmd.Stdout = &buf
			if err := cmd.Run(); err != nil {
				log.Printf("Failed to get process snapshot: %v", err)
				return
			}

			// push measured info to output channel
			var data interface{}
			if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
				log.Printf("Failed to unmarshal %s: %v", buf.Bytes(), err)
				return
			}
			buf.Reset()
			m := data.(map[string]interface{})
			m["node"] = ag.me.Ip
			m["app_id"] = app.Id
			docCh <- &Doc{kind: DocTypeProc, content: m}
		}
	}

	go func() {
		defer close(docCh)
		for {
			for _, app := range ag.getMoniList() {
				moniOne(app)
			}
			time.Sleep(ag.moniItv)
		}
	}()
	return docCh
}

// sysMonitor measures system resource usage periodly
func (ag *Agent) sysMonitor() <-chan *Doc {
	docCh := make(chan *Doc, 5)
	go func() {
		defer close(docCh)
		var buf bytes.Buffer
		for {
			cmd := exec.Command("python",
				path.Join(dmoniPath, "/snapshot/sysusage.py"))
			cmd.Stdout = &buf
			if err := cmd.Run(); err != nil {
				log.Printf("Failed to get system snapshot: %v", err)
				continue
			}

			// push measured info to output channel
			var data interface{}
			if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
				log.Printf("Failed to unmarshal %s: %v", buf.Bytes(), err)
				continue
			}
			buf.Reset()
			m := data.(map[string]interface{})
			m["node"] = ag.me.Ip
			docCh <- &Doc{kind: DocTypeSys, content: m}

			time.Sleep(ag.moniItv)
		}
	}()
	return docCh
}

// merger receives docs from multiple channels and sends
// them to a single out channel.
func (ag *Agent) merger(docChs ...<-chan *Doc) chan *Doc {
	outCh := make(chan *Doc, 5)
	var wg sync.WaitGroup
	wg.Add(len(docChs))

	for _, ch := range docChs {
		go func(c <-chan *Doc) {
			defer wg.Done()
			for doc := range c {
				outCh <- doc
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(outCh)
	}()
	return outCh
}

// Detect an application's processes
func (ag *Agent) detectProcs(app *App) []common.Process {
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
			log.Printf("Failed to detect processes: %s", app.Id, err)
			continue
		}
		procs = append(procs, fps...)
	}
	return procs
}

// launch runs an application as a subprocess.
func (ag *Agent) launch(lch chan *AppExec) {
	go func() {
		for e := range ag.lch {
			app := &App{
				Id:   e.appId,
				exe:  e.cmd,
				args: e.args,
				sout: bytes.NewBuffer(make([]byte, 0, 1024)),
				serr: bytes.NewBuffer(make([]byte, 0, 1024)),
			}

			// Run application as a child process
			app.ctx, app.cancel = context.WithCancel(context.Background())
			cmd := exec.CommandContext(app.ctx, app.exe, app.args...)
			cmd.Stdout = app.sout
			cmd.Stderr = app.serr
			err := cmd.Start()
			if err != nil {
				log.Printf("Failed to execute %s %v: %v", app.exe, app.args, err)
			}
			// launching is done, return err
			e.errCh <- err

			app.stime = time.Now()
			app.EntryPid = cmd.Process.Pid

			ag.lchApps.Lock()
			ag.lchApps.m[app.Id] = app
			ag.lchApps.Unlock()

			docId, _ := ag.storeOne(&Doc{kind: DocTypeApp,
				content: map[string]interface{}{
					"app_id":     app.Id,
					"entry_node": ag.me.Ip,
					"exec":       app.exe,
					"args":       app.args,
					"start_at":   app.stime.Format(time.RFC3339),
					"timestamp":  time.Now().Format(time.RFC3339),
				}})

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
					// Application is killed by manager
					log.Printf("App %s was killed", app.Id)
					return
				default:
					app.etime = time.Now()
					ag.updateDoc(docId, &Doc{kind: DocTypeApp,
						content: map[string]interface{}{
							"stdout": app.sout.String(),
							"stderr": app.serr.String(),
							"end_at": app.etime.Format(time.RFC3339),
						}})

					ag.notifyDone(app)
					ag.lchApps.Lock()
					delete(ag.lchApps.m, app.Id)
					ag.lchApps.Unlock()
				}
			}()
		}
	}()
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
		time.Sleep(ag.hbItv)
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
			time.Sleep(ag.hbItv)
			go ag.cast()
			return
		}

		// Update manager's info
		ag.manager.Id = ma.Id
		ag.manager.Heartbeat = ma.Heartbeat

		time.Sleep(ag.hbItv)
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

// updateDoc update an ElasticSearch docment given doc id.
func (ag *Agent) updateDoc(id string, doc *Doc) error {
	// Create an ElasticSearch client
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL(ag.dsAddr))
	if err != nil {
		log.Printf("Failed to create ElasticSearch client: %v", err)
		return err
	}

	// Push doc
	_, err = client.Update().Index("dmoni").Type(doc.kind).
		Id(id).Doc(doc.content).Refresh(true).Do()
	if err != nil {
		log.Printf("Failed to update document: %v", err)
		return err
	}
	return nil
}

// storeOne pushes a doc to ElasticSearch immediately.
func (ag *Agent) storeOne(doc *Doc) (docId string, err error) {
	// Create an ElasticSearch client
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL(ag.dsAddr))
	if err != nil {
		log.Printf("Failed to create ElasticSearch client: %v", err)
		return "", err
	}

	// Push doc
	resp, err := client.Index().Index("dmoni").Type(doc.kind).
		BodyJson(doc.content).Refresh(true).Do()
	if err != nil {
		log.Printf("Failed to index document: %v", err)
		return "", err
	}
	docId = resp.Id
	return docId, err
}

// store pulls data from a channel and pushes them into ElasticSearch
func (ag *Agent) store(docCh <-chan *Doc) {
	go func() {
		// Create an ElasticSearch client
		client, err := elastic.NewClient(
			elastic.SetSniff(false),
			elastic.SetURL(ag.dsAddr))
		if err != nil {
			log.Fatalf("Failed to create ElasticSearch client: %v", err)
		}

		// Check the existence of index; if not create one
		// TODO(lizhong): move it to checkConfig()
		exist, err := client.IndexExists("dmoni").Do()
		if err != nil {
			log.Fatalf("Failed to call IndexExists: %v", err)
		}
		if !exist {
			_, err = client.CreateIndex("dmoni").Do()
			if err != nil {
				log.Fatalf("Failed to create index: %v", err)
			}
		}

		// Creat a BulkProcessor in order to push docs in batch
		bulk, err := client.BulkProcessor().
			BulkActions(1).BulkSize(5 * 1024 * 1024).
			FlushInterval(time.Minute * 5).Do()
		if err != nil {
			log.Fatalf("Failed to create BulkProcessor")
		}
		defer bulk.Close()

		for doc := range docCh {
			req := elastic.NewBulkIndexRequest().
				Index("dmoni").Type(doc.kind).Doc(doc.content)
			bulk.Add(req)
		}
	}()
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

// getMoniList returns a list of applications to be monitored
func (ag *Agent) getMoniList() []*App {
	ag.apps.RLock()
	defer ag.apps.RUnlock()
	apps := make([]*App, 0, len(ag.apps.m))
	for _, a := range ag.apps.m {
		apps = append(apps, a)
	}
	return apps
}
