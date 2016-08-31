package main

import (
	//"fmt"
	"log"

	"github.com/lizhongz/dmoni/agent"
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

	ag, err := agent.NewAgent()
	if err != nil {
		log.Fatal(err)
	}

	app := agent.App{
		Id:         "",
		Frameworks: []string{"hadoop", "spark"},
	}

	ag.Register(&app)
	ag.Monitor()
}
