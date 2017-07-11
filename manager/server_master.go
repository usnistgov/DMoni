package manager

import (
	"fmt"
	"log"
	"net"
	"time"

	//"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/usnistgov/DMoni/common"
	mPb "github.com/usnistgov/DMoni/proto/manager"
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
		grpclog.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	mPb.RegisterManagerServer(grpcServer, s)
	grpcServer.Serve(lis)
}

// SayHi registers an agent or updates its info, particularly heartbeat.
func (s *masterServer) SayHi(ctx context.Context, ni *mPb.NodeInfo) (*mPb.NodeInfo, error) {
	grpclog.Printf("Hi from agent at %s:%d", ni.Host, ni.Port)

	ag := s.mng.getAgent(ni.Id)
	if ag == nil {
		// New agent
		ag = &common.Node{
			Id:   ni.Id,
			Host: ni.Host,
			Port: ni.Port,
		}
		s.mng.agents.Lock()
		s.mng.agents.m[ni.Id] = ag
		s.mng.agents.Unlock()
		grpclog.Printf("New agent %s at %s:%d", ni.Id, ni.Host, ni.Port)

		go func() {
			// TODO(lizhong): bug - if an agent exits and rejoins with the same id,
			// this will not be triggered.

			// Send configurations to agent
			newCtx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()
			s.mng.configAgent(newCtx, ag)
		}()
	}

	ag.Timestamp = time.Now()

	// Return my node info
	return &mPb.NodeInfo{
		Id:        s.mng.me.Id,
		Host:      s.mng.me.Host,
		Port:      s.mng.me.Port,
		Heartbeat: s.mng.me.Heartbeat,
	}, nil
}

// NotifyDone signifies the finish of an applicaiton and
// triggers application deregistration.
func (s *masterServer) NotifyDone(ctx context.Context, in *mPb.NDRequest) (*mPb.NDReply, error) {
	log.Printf("App %s exited", in.AppId)

	app := s.mng.getApp(in.AppId)
	if app.monitored {
		// Stop monitoring the app on agents
		newCtx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()
		err := s.mng.deregister(newCtx, app, true)
		if err != nil {
			log.Printf("Failed to deregister app %s: %v", in.AppId, err)
		}
	}

	s.mng.apps.Lock()
	delete(s.mng.apps.m, in.AppId)
	s.mng.apps.Unlock()

	return &mPb.NDReply{}, nil
}
