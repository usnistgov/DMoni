package manager

import (
	"fmt"
	"net"
	"time"

	//"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/lizhongz/dmoni/common"
	cPb "github.com/lizhongz/dmoni/proto/cluster"
)

var ()

type masterServer struct {
	mng *Manager
}

func newMasterServer(mng *Manager) *masterServer {
	s := new(masterServer)
	s.mng = mng
	return s
}

func (s *masterServer) Run() {
	// Run clustering server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.mng.me.Port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	cPb.RegisterMembershipServer(grpcServer, s)
	grpcServer.Serve(lis)
}

// SayHi registers an agent or updates its info, particularly heartbeat.
func (s *masterServer) SayHi(ctx context.Context, ni *cPb.NodeInfo) (*cPb.NodeInfo, error) {
	if _, present := s.mng.agents[ni.Id]; !present {
		// Create a new agent
		s.mng.agents[ni.Id] = &common.Node{Id: ni.Id}
		grpclog.Printf("New agent %s %s:%d", ni.Id, ni.Ip, ni.Port)
	}

	// Update agent's node info
	n := s.mng.agents[ni.Id]
	n.Ip = ni.Ip
	n.Port = ni.Port
	n.Heartbeat = ni.Heartbeat
	n.Timestamp = time.Now()

	//grpclog.Printf("Hi from %s", n.Id)

	// Return my node info
	return &cPb.NodeInfo{
		Id:        s.mng.me.Id,
		Ip:        s.mng.me.Ip,
		Port:      s.mng.me.Port,
		Heartbeat: s.mng.me.Heartbeat,
	}, nil
}
