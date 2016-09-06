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
	me     common.Node
	agents map[string]*common.Node
}

func newServer(id string, ip string, port int32) *managerServer {
	s := new(managerServer)
	s.me = common.Node{
		Id:        id,
		Ip:        ip,
		Port:      port,
		Heartbeat: 0,
		Timestamp: time.Now(),
	}
	s.agents = make(map[string]*common.Node)
	return s
}

func (s *managerServer) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.me.Port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterMembershipServer(grpcServer, s)
	grpcServer.Serve(lis)
}

// SayHi registers an agent or updates its info, particularly heartbeat.
func (s *managerServer) SayHi(ctx context.Context, ni *pb.NodeInfo) (*pb.NodeInfo, error) {
	if _, present := s.agents[ni.Id]; !present {
		// Create a new agent
		s.agents[ni.Id] = &common.Node{Id: ni.Id}
	}

	// Update agent's node info
	n := s.agents[ni.Id]
	n.Ip = ni.Ip
	n.Port = ni.Port
	n.Heartbeat = ni.Heartbeat
	n.Timestamp = time.Now()

	grpclog.Printf("Hi from %s", n.Id)

	// Return my node info
	return &pb.NodeInfo{
		Id:        s.me.Id,
		Ip:        s.me.Ip,
		Port:      s.me.Port,
		Heartbeat: s.me.Heartbeat,
	}, nil
}
