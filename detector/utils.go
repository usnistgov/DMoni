package detector

import (
	"bytes"
	"os/exec"
	"strings"
)

// Execute command line tool jps to retrive java applications
func cmd_jps() ([]Process, error) {
	// Execute command
	cmd := exec.Command("jps", "-lm")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	// Retrieve process info: pid and command
	procs := []Process{}
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			procs = append(procs, Process{
				Pid:      parts[0],
				FullName: parts[1],
			})
		}
	}
	return procs, nil
}
