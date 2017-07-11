// Copyright © 2016 Lizhong Zhang <lizhong.zhang@nist.gov>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"log"
	"os"

	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"

	"github.com/usnistgov/DMoni/agent"
)

// Define flags
var (
	// Agent's address
	agHost string
	// Agent's port
	agPort int
	// Manager's address
	agMng string
	// Manager's port
	agMngPort int
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run Dmoni agent.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(agHost) == 0 {
			// If host not given, using hostname instead
			var err error
			agHost, err = os.Hostname()
			if err != nil {
				log.Fatal("Failed to get hostname")
			}
		}

		ag := agent.NewAgent(
			&agent.Config{
				Id:      uuid.NewV4().String(),
				Host:    agHost,
				Port:    int32(agPort),
				Mng:     agMng,
				MngPort: int32(agMngPort),
				DsAddr:  dsAddr,
			})
		ag.Run()
	},
}

func init() {
	RootCmd.AddCommand(agentCmd)

	agentCmd.Flags().StringVarP(&agHost, "host", "", "", "Host's address (hostname, IP address, etc.). (default: hostname)")
	agentCmd.Flags().IntVarP(&agPort, "port", "p", 5301, "Agent's port.")
	agentCmd.Flags().StringVarP(&agMng, "manager", "m", "", "Manager's address.")
	agentCmd.Flags().IntVarP(&agMngPort, "manager-port", "", 5300, "Manager's port.")
}
