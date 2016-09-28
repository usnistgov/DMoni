package main

import (
	//"fmt"
	"flag"
	"log"

	"github.com/lizhongz/dmoni/agent"
	"github.com/lizhongz/dmoni/manager"
)

func main() {
	mRole := flag.Bool("manager", false, "The role of the monitoring deamon")
	flag.Parse()

	if *mRole {
		// Acting as a manager
		log.Printf("Dmoni manager")
		m := manager.NewManager(
			&manager.Config{
				Id:       "manager",
				NodePort: 5300,
				AppPort:  5500,
			})
		m.Run()
	} else {
		// Acting as an agent
		log.Printf("Dmoni agent")
		ag := agent.NewAgent(
			&agent.Config{
				Id:      "agent-0",
				Ip:      "192.168.0.6",
				Port:    5301,
				MngIp:   "192.168.0.6",
				MngPort: 5300,
			})
		ag.Run()
	}

}
