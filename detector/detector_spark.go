package detector

import (
	"strings"

	"github.com/lizhongz/dmoni/common"
)

type SparkDetector struct{}

// Spark framework processes
var sparkProcMap = map[string]string{
	"Master":    "org.apache.spark.deploy.master.Master",
	"Worker":    "org.apache.spark.deploy.worker.Worker",
	"HisServer": "org.apache.spark.deploy.history.HistoryServer",
}

// Spark application related processes
var sparkAppProcMap = map[string]string{
	// TODO(lizhong): recognize Submit process
	//"SparkSubmit": "org.apache.spark.deploy.SparkSubmit",
	"CoarseGrainedExecutorBackend": "org.apache.spark.executor.CoarseGrainedExecutorBackend",
}

var sparkProcPrefix = "org.apache.spark"

func (d SparkDetector) Detect(jobId string) ([]common.Process, error) {
	// Obtain all java processes
	procs, err := cmd_jps()
	if err != nil {
		return nil, err
	}

	sProcs := []common.Process{}

	// Filter Spark and application processes
	for _, proc := range procs {
		if !strings.Contains(proc.FullName, sparkProcPrefix) {
			// Not a Spark process
			continue
		}

		// Obtain process' short name
		spl := strings.Split(strings.SplitN(proc.FullName, " ", 2)[0], ".")
		sn := spl[len(spl)-1]

		valid := false
		if _, ok := sparkProcMap[sn]; ok {
			// Spark process
			valid = true
		} else if _, ok := sparkAppProcMap[sn]; ok {
			// Application process
			if strings.Contains(proc.FullName, jobId) {
				valid = true
			}
		} else {
			// Unknown Spark process
			valid = true
		}

		if valid {
			sProcs = append(sProcs, common.Process{
				Pid:       proc.Pid,
				ShortName: sn,
				// Full name including command arguments
				FullName: proc.FullName})
		}

	}

	return sProcs, nil
}
