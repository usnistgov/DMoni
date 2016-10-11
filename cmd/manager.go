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
	"github.com/lizhongz/dmoni/manager"

	"github.com/spf13/cobra"
)

// Define flags
var mAppPort int
var mNodePort int
var mId string

// managerCmd represents the manager command
var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Run DMoni as a manager.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		m := manager.NewManager(
			&manager.Config{
				Id:       mId,
				NodePort: int32(mNodePort),
				AppPort:  int32(mAppPort),
				DsAddr:   dsAddr,
			})
		m.Run()
	},
}

func init() {

	RootCmd.AddCommand(managerCmd)

	managerCmd.Flags().IntVarP(&mAppPort, "app-port", "", 5500, "Port used to talk with dmoni application clients.")
	managerCmd.Flags().IntVarP(&mNodePort, "node-port", "", 5300, "Port used to talk with dmoni agents.")
	managerCmd.Flags().StringVarP(&mId, "id", "", "manager", "Name or identity for manager.")
}
