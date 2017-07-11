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

	"github.com/spf13/cobra"

	"github.com/usnistgov/DMoni/monica"
)

// Flags
var (
	host       string
	frameworks []string
	perf       bool
)

// submitCmd represents the submit command
var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Launch an application and start to monitor it with dmoni.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		monica.SetConfig(monica.Config{
			DmoniAddr:   managerAddr,
			StorageAddr: dsAddr,
		})

		if len(host) == 0 {
			var err error
			host, err = os.Hostname()
			if err != nil {
				log.Fatal("Failed to get hostname")
			}
		}
		log.Print(host)

		appSub := &monica.AppSub{
			Entry:      host,
			Cmd:        args[0],
			Frameworks: frameworks,
			Perf:       perf,
		}
		if len(args) > 1 {
			appSub.Args = args[1:]
		}
		monica.Submit(appSub)
	},
}

func init() {
	MonicaCmd.AddCommand(submitCmd)

	submitCmd.Flags().StringVarP(&host, "host", "", "", "The host where the application will be launched (default: localhost's hostname)")
	submitCmd.Flags().StringSliceVarP(&frameworks, "frameworks", "", []string{}, "Frameworks used, e.g. hadoop spark")
	submitCmd.Flags().BoolVarP(&perf, "perf", "", false, "Enables monitoring applications performance metrics")
}
