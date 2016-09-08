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
	pb "github.com/lizhongz/dmoni/proto/cluster"
)

var ()

type managerServer struct {
	mng *Manager
}

func newServer(mng *Manager) *managerServer {
	s := new(managerServer)
	s.mng = mng
	return s
}

func (s *managerServer) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.mng.me.Port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterMembershipServer(grpcServer, s)
	grpcServer.Serve(lis)
}

// SayHi registers an agent or updates its info, particularly heartbeat.
func (s *managerServer) SayHi(ctx context.Context, ni *pb.NodeInfo) (*pb.NodeInfo, error) {
	if _, present := s.mng.agents[ni.Id]; !present {
		// Create a new agent
		s.mng.agents[ni.Id] = &common.Node{Id: ni.Id}
	}

	// Update agent's node info
	n := s.mng.agents[ni.Id]
	n.Ip = ni.Ip
	n.Port = ni.Port
	n.Heartbeat = ni.Heartbeat
	n.Timestamp = time.Now()

	//grpclog.Printf("Hi from %s", n.Id)

	// Return my node info
	return &pb.NodeInfo{
		Id:        s.mng.me.Id,
		Ip:        s.mng.me.Ip,
		Port:      s.mng.me.Port,
		Heartbeat: s.mng.me.Heartbeat,
	}, nil
}
