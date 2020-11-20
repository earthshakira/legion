package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sophron-dev-works/legion/clustering"
	"github.com/sophron-dev-works/legion/dbshell"
	"github.com/sophron-dev-works/legion/execution"
	"github.com/sophron-dev-works/legion/storage"
	"github.com/sophron-dev-works/legion/ui"
)

var raftPort, serfPort, httpPort, grpcPort int
var ip, joinNodeID, raftDir string

func init() {
	flag.IntVar(&httpPort, "http", 8003, "HTTP Port")
	flag.IntVar(&serfPort, "serf", 8001, "Serf Port")
	flag.IntVar(&raftPort, "raft", 8002, "Raft Port")
	flag.IntVar(&grpcPort, "grpc", 8004, "grpc Port")
	flag.StringVar(&ip, "ip", "localhost", "LocalIp")
	flag.StringVar(&joinNodeID, "join", "", "id of a node to join")
	flag.StringVar(&raftDir, "raftDir", "./raftStorage", "Directory to store Raft Logs")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func startUI() ui.IrisWrapper {

	nc := clustering.NodeConfig{
		Ip:       ip,
		HttpPort: httpPort,
		SerfPort: serfPort,
		RaftPort: raftPort,
		GrpcPort: grpcPort,
		RaftDir:  raftDir,
		Join:     joinNodeID,
	}

	nc.Init()

	var node clustering.Node
	node.Init(&nc)
	go node.Start()

	var db storage.SimpleStore
	db.Init(&node)

	var ee execution.ExecutionEngine
	ee.Init(&node)

	var ex execution.Executor
	ex.Init(&node, &nc, &db)
	go ex.Serve()
	go ex.Listen()
	fmt.Println("done")

	iw := ui.Init()
	var dash clustering.Dashboard
	dash.Init(&node)

	var we execution.WorkflowEngine
	we.Init(&node, &ex)

	var se dbshell.ShellEngine
	se.SetNode(db, &node)

	var ds dbshell.DistributedShell
	go ds.Listen()
	ds.Init(db, &node)

	iw.AddApp("/dashboard", &dash)
	iw.AddApp("/scripts", &ee)
	iw.AddApp("/workflows", &we)
	iw.AddApp("/shell", &se)
	iw.AddApp("/dshell", &ds)
	iw.Start(httpPort)
	return iw
}

func main() {
	flag.Parse()
	iw := startUI()
	defer iw.Clean()
}
