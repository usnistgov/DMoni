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
	"net"

	"github.com/lizhongz/dmoni/agent"

	"github.com/spf13/cobra"
)

var aId string  // Agent's ID
var aIP net.IP  // Agent's IP address
var aPort int   // Agent's port
var aMIP net.IP // Manager's IP address
var aMPort int  // Manager's port

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run DMoni as an agent.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ag := agent.NewAgent(
			&agent.Config{
				Id:      aId,
				Ip:      aIP.String(),
				Port:    int32(aPort),
				MngIp:   aMIP.String(),
				MngPort: int32(aMPort),
				DsAddr:  dsAddr,
			})
		ag.Run()
	},
}

func init() {
	RootCmd.AddCommand(agentCmd)

	agentCmd.Flags().StringVarP(&aId, "id", "", "", "Name or identity for agent.")
	agentCmd.Flags().IPVarP(&aIP, "ip", "", nil, "Agent's ip address.")
	agentCmd.Flags().IntVarP(&aPort, "port", "", 5301, "Agent's port.")
	agentCmd.Flags().IPVarP(&aMIP, "mip", "", nil, "Manager's ip address.")
	agentCmd.Flags().IntVarP(&aMPort, "mport", "", 5300, "Manager's port.")
}
