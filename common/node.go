package common

import (
	"time"
)

type Node struct {
	// Unique identity
	Id string
	// Node's address
	Host string
	Port int32
	// Node's heartbeat counter
	Heartbeat int32
	// The timestamp when heartbeat counter was updated
	Timestamp time.Time
}
