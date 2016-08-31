package agent

import (
	"bytes"
	"log"
	"os/exec"
	"time"

	"github.com/lizhongz/dmoni/detector"
)

// Application info used for monitoring
type App struct {
	// Application Id
	Id string
	// Frameworks used by this application
	Frameworks []string
}

type Agent struct {
	// List of monitored applications
	apps map[string]App
	// Process detectors for different frameworks
	// (key, value) = (framework, detector)
	detectors map[string]detector.Detector
	// Application processes
	// (key, value) = (app id, process list)
	appProcs map[string][]detector.Process
}

func NewAgent() (*Agent, error) {
	ag := &Agent{
		apps: make(map[string]App),
		detectors: map[string]detector.Detector{
			"hadoop": detector.HadoopDetector{},
			"spark":  detector.SparkDetector{},
		},
		appProcs: make(map[string][]detector.Process),
	}

	return ag, nil
}

// Register an application
func (ag *Agent) Register(app *App) {
	fws := make([]string, len(app.Frameworks))
	copy(fws, app.Frameworks)
	log.Print("copied")
	ag.apps[app.Id] = App{
		Id:         app.Id,
		Frameworks: fws,
	}
}

// Unregister an application
func (ag *Agent) unregister(appId string) {
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
					"/home/lnz5/workspace/snapshot/app/monitor.py", "-n", "1", p.Pid)
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
