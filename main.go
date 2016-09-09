package main

import (
	//"fmt"
	"flag"
	"log"

	"github.com/lizhongz/dmoni/agent"
	//"github.com/lizhongz/dmoni/common"
	"github.com/lizhongz/dmoni/manager"
	//"github.com/lizhongz/dmoni/detector"
)

func main() {
	/*
		var hd detector.HadoopDetector
		hProcs, err := hd.Detect("")
		if err != nil {
			log.Fatal(err)
		}

		for _, p := range hProcs {
			fmt.Println(p)
		}
	*/

	/*
		var sd detector.SparkDetector
		sProcs, err := sd.Detect("")
		if err != nil {
			log.Fatal(err)
		}

		for _, p := range sProcs {
			fmt.Println(p)
		}
	*/

	/*
		ag, err := agent.NewAgent()
		if err != nil {
			log.Fatal(err)
		}

		app := common.App{
			Id:         "",
			Frameworks: []string{"hadoop", "spark"},
		}

		ag.Register(&app)
		ag.Monitor()
	*/

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
