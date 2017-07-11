package detector

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"github.com/usnistgov/DMoni/common"
)

// Execute command line tool jps to retrive java applications
func cmd_jps() ([]common.Process, error) {
	// Execute command
	cmd := exec.Command("jps", "-lm")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// Retrieve process info: pid and command
	procs := []common.Process{}
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, err
			}
			procs = append(procs, common.Process{
				Pid:      pid,
				FullName: parts[1],
			})
		}
	}
	return procs, nil
}
