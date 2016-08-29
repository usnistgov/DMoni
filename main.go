package main

import (
	"fmt"
	"github.com/lizhongz/dmoni/detector"
	"log"
)

func main() {
	var hd detector.HadoopDetector
	hProcs, err := hd.Detect("")
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range hProcs {
		fmt.Println(p)
	}

	var sd detector.SparkDetector
	sProcs, err := sd.Detect("")
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range sProcs {
		fmt.Println(p)
	}
}
