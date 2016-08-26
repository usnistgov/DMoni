package main

import (
	"fmt"
	"github.com/lizhongz/dmoni/detector"
	"log"
)

func main() {
	var hd detector.HadoopDetector
	procs, err := hd.Detect("asdf")
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range procs {
		fmt.Println(p)
	}
}
