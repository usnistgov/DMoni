package common

import (
	"time"
)

type Node struct {
	Id        string
	Ip        string
	Port      int32
	Heartbeat int32
	Timestamp time.Time
}
