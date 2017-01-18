package monica

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/olivere/elastic.v3"

	pb "github.com/lizhongz/dmoni/proto/app"
)

var (
	dmoniAddr string
	storeAddr string
)

type AppSub struct {
	Entry      string
	Cmd        string
	Args       []string
	Frameworks []string
	Perf       bool
}

type Config struct {
	DmoniAddr   string // DMoni server's address
	StorageAddr string // Storage's address
}

func SetConfig(cfg Config) {
	dmoniAddr = cfg.DmoniAddr
	storeAddr = cfg.StorageAddr
}

// Submit launches an application using dmoni.
func Submit(app *AppSub) {
	client, close, err := newAppGaugeClient()
	if err != nil {
		fmt.Println("Unable to connect to AppGaugeServer: %v", err)
		os.Exit(-1)
	}
	defer close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	reply, err := client.Submit(ctx, &pb.SubRequest{
		EntryNode:  app.Entry,
		ExecName:   app.Cmd,
		ExecArgs:   app.Args,
		Frameworks: app.Frameworks,
		Moni:       app.Perf,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to submit app: %v", err)
		os.Exit(-1)
	}
	fmt.Printf("Application Id: %s", reply.Id)
	os.Exit(0)
}

// Kill stops an running application.
func Kill(appId string) {
	client, close, err := newAppGaugeClient()
	if err != nil {
		fmt.Println("Unable to connect to AppGaugeServer: %v", err)
		os.Exit(-1)
	}
	defer close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = client.Kill(ctx, &pb.AppIndex{Id: appId})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to kill app %s: %v", appId, err)
		os.Exit(-1)
	}
}

// AppInfo returns an app's basic information.
func AppInfo(appId string) {
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL(storeAddr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create ElasticSearch client: %v", err)
		os.Exit(-1)
	}

	termQuery := elastic.NewTermQuery("app_id", appId)
	resp, err := client.Search().
		Index("dmoni").
		Type("app").
		Query(termQuery).
		Pretty(true).
		Do()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query app: %v", err)
		os.Exit(-1)
	}

	if resp.Hits.TotalHits == 0 {
		fmt.Fprintf(os.Stderr, "App does not exist")
		os.Exit(0)
	}
	for _, item := range resp.Hits.Hits {
		doc, err := item.Source.MarshalJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to jsonfy resutls")
		}
		fmt.Printf("%s\n", string(doc))
	}
}

// AppPerf returns measured performance data of a given app.
func AppPerf(appId string) {
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL(storeAddr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create ElasticSearch client: %v", err)
		os.Exit(-1)
	}

	termQuery := elastic.NewTermQuery("app_id", appId)
	scroll := client.Scroll().
		Index("dmoni").
		Type("proc").
		Query(termQuery).
		Sort("timestamp", false). // Descending
		Size(100).
		Pretty(true)

	for {
		resp, err := scroll.Do()
		if err == io.EOF {
			os.Exit(0)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query app: %v", err)
			os.Exit(-1)
		}

		for _, item := range resp.Hits.Hits {
			doc, err := item.Source.MarshalJSON()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to jsonfy resutls")
			}
			fmt.Printf("%s\n", string(doc))
		}
	}
}

// AppPerf returns measured system performance of a given app.
func AppSysPerf(appId string) {
	client, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetURL(storeAddr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create ElasticSearch client: %v", err)
		os.Exit(-1)
	}

	// Query app's start time and end time
	termQuery := elastic.NewTermQuery("app_id", appId)
	resp, err := client.Search().
		Index("dmoni").
		Type("app").
		Query(termQuery).
		Fields("start_at", "end_at").
		Pretty(true).
		Do()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query app: %v", err)
		os.Exit(-1)
	}
	if resp.Hits.TotalHits == 0 {
		fmt.Fprintf(os.Stderr, "App does not exist")
		os.Exit(-1)
	}
	fs := resp.Hits.Hits[0].Fields
	sf, ok := fs["start_at"]
	if !ok {
		fmt.Fprintf(os.Stderr, "start time does not exist")
		os.Exit(-1)
	}
	ef, ok := fs["end_at"]
	if !ok {
		fmt.Fprintf(os.Stderr, "end time does not exist")
		os.Exit(-1)
	}

	// Filter system's performance between start time and end time
	st, err := time.Parse(time.RFC3339, sf.([]interface{})[0].(string))
	et, err := time.Parse(time.RFC3339, ef.([]interface{})[0].(string))
	st = st.Add(time.Minute * (-3))
	et = et.Add(time.Minute * 3)

	rq := elastic.NewRangeQuery("timestamp").
		Gte(st).
		Lte(et)

	scroll := client.Scroll().
		Index("dmoni").
		Type("sys").
		Query(rq).
		Sort("timestamp", false). // decsending
		Size(100).
		Pretty(true)

	for {
		resp, err = scroll.Do()
		if err == io.EOF {
			os.Exit(0)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to query system performance: %v", err)
			os.Exit(-1)
		}
		if resp.Hits.TotalHits == 0 {
			fmt.Fprintf(os.Stderr, "No system performance info found")
			os.Exit(-1)
		}

		// Print system performance info
		for _, item := range resp.Hits.Hits {
			doc, err := item.Source.MarshalJSON()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to jsonfy resutls")
			}
			fmt.Printf("%s\n", string(doc))
		}
	}
}

// GetProcesses pull application's processes from AppGaugeServer.
func GetProcesses(appId string) {
	client, close, err := newAppGaugeClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to AppGaugeServer: %v", err)
		os.Exit(-1)
	}
	defer close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	reply, err := client.GetProcesses(ctx, &pb.AppIndex{Id: appId})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get app %s's processes: %v", appId, err)
		os.Exit(-1)
	}
	for node, list := range reply.NodeProcs {
		fmt.Printf("Node %s\n", node)
		for _, p := range list.Procs {
			fmt.Printf("  %d %s (%s)\n", p.Pid, p.Name, p.Cmd)
		}
	}
	os.Exit(0)
}

// newRPCClient create an client connecting to dmoni AppGaugeServer.
func newAppGaugeClient() (client pb.AppGaugeClient, close func(), err error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(dmoniAddr, opts...)
	if err != nil {
		grpclog.Fatalf("Failed to dial AppGuageServer: %v", err)
		return nil, nil, err
	}
	client = pb.NewAppGaugeClient(conn)
	return client, func() { conn.Close() }, nil
}
