package detector

import (
	"strings"
)

type HadoopDetector struct{}

// Hadoop framework processes
var hadoopProcMap = map[string]string{
	"ResourceManager":   "org.apache.hadoop.yarn.server.resourcemanager.ResourceManager",
	"DataNode":          "org.apache.hadoop.hdfs.server.datanode.DataNode",
	"NameNode":          "org.apache.hadoop.hdfs.server.namenode.NameNode",
	"SecondaryNameNode": "org.apache.hadoop.hdfs.server.namenode.SecondaryNameNode",
	"NodeManager":       "org.apache.hadoop.yarn.server.nodemanager.NodeManager",
	"JobHistoryServer":  "org.apache.hadoop.mapreduce.v2.hs.JobHistoryServer",
}

// Hadoop application related processes
var hadoopAppProcMap = map[string]string{
	// TODO(lizhong): recognize RunJar process
	//"RunJar":      "org.apache.hadoop.util.RunJar",
	"MRAppMaster": "org.apache.hadoop.mapreduce.v2.app.MRAppMaster",
	"YarnChild":   "org.apache.hadoop.mapred.YarnChild",
}

var hadoopProcPrefix = "org.apache.hadoop"

func (d HadoopDetector) Detect(appId string) ([]Process, error) {
	// Obtain all java processes
	procs, err := cmd_jps()
	if err != nil {
		return nil, err
	}

	haProcs := []Process{}

	// Filter hadoop and application processes
	for _, proc := range procs {
		if !strings.Contains(proc.FullName, hadoopProcPrefix) {
			// Not a hadoop process
			continue
		}

		// Obtain process' short name
		spl := strings.Split(strings.SplitN(proc.FullName, " ", 2)[0], ".")
		sn := spl[len(spl)-1]

		valid := false
		if _, ok := hadoopProcMap[sn]; ok {
			// Hadoop process
			valid = true
		} else if _, ok := hadoopAppProcMap[sn]; ok {
			// Application process
			if strings.Contains(proc.FullName, appId) {
				valid = true
			}
		} else {
			// Unknown Hadoop process
			valid = true
		}

		if valid {
			haProcs = append(haProcs, Process{
				Pid:       proc.Pid,
				ShortName: sn,
				// Full name including command arguments
				FullName: proc.FullName})
		}

	}

	return haProcs, nil
}
