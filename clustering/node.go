package clustering

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/serf/serf"
)

type NodeConfig struct {
	HttpPort int
	Id       string
	Ip       string
	SerfPort int
	RaftPort int
	GrpcPort int
	RaftDir  string
	Join     string
}

func (nc *NodeConfig) Init() {
	nc.Id = fmt.Sprintf("%s:%d:%d", nc.Ip, nc.SerfPort, nc.RaftPort)
}

func (nc *NodeConfig) SerfPeer() string {
	if nc.Join == "" {
		return ""
	}
	p := strings.Split(nc.Join, ":")
	return p[0] + ":" + p[1]
}

func (nc *NodeConfig) RaftPeer() string {
	if nc.Join == "" {
		return ""
	}
	p := strings.Split(nc.Join, ":")
	return p[0] + ":" + p[2]
}

type Node struct {
	Id           string
	serf         *serf.Serf
	store        *RaftStore
	serfEvents   chan serf.Event
	Queries      chan *serf.Query
	UserEvents   chan *serf.UserEvent
	GRPCEndpoint string
}

func (node *Node) Events() chan serf.Event {
	return node.serfEvents
}

func (node *Node) Serf() *serf.Serf {
	return node.serf
}

func (node *Node) Raft() *RaftStore {
	return node.store
}

func (node *Node) serfInit(n *NodeConfig) {
	fmt.Println("NodeID", n.Id)
	node.Id = n.Id
	var err error
	node.serfEvents = make(chan serf.Event, 16)
	memberlistConfig := memberlist.DefaultLANConfig()
	memberlistConfig.BindAddr = n.Ip
	memberlistConfig.BindPort = n.SerfPort
	memberlistConfig.LogOutput = os.Stdout
	serfConfig := serf.DefaultConfig()
	serfConfig.NodeName = n.Id
	serfConfig.EventCh = node.serfEvents
	serfConfig.MemberlistConfig = memberlistConfig
	serfConfig.LogOutput = os.Stdout

	serfConfig.Tags = make(map[string]string)
	serfConfig.Tags["httpPort"] = fmt.Sprintf("%d", n.HttpPort)
	serfConfig.Tags["raft"] = fmt.Sprintf("%d", n.RaftPort)
	serfConfig.Tags["grpc"] = fmt.Sprintf("%s:%d", n.Ip, n.GrpcPort)
	node.GRPCEndpoint = fmt.Sprintf("%s:%d", n.Ip, n.GrpcPort)
	node.serf, err = serf.Create(serfConfig)
	if err != nil {
		log.Fatal(err)
	}

	if n.Join != "" {
		_, err = node.serf.Join([]string{n.SerfPeer()}, false)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("Serf is up for ", n.Id)
}

func (node *Node) raftInit(n *NodeConfig) {
	inmem := false
	os.MkdirAll(n.RaftDir, 0700)
	node.store = New(inmem)
	node.store.RaftDir = n.RaftDir
	node.store.RaftBind = n.Ip + fmt.Sprintf(":%d", n.RaftPort)
	err := node.store.Open(n.Join == "", n.Id)
	fmt.Println("RaftStore is open ", n.Id, err)
}

func (node *Node) JoinRaft(id string) {
	p := strings.Split(id, ":")
	node.store.Join(id, p[0]+":"+p[2])
}

func (node *Node) LeaveRaft(id string) {
	p := strings.Split(id, ":")
	err := node.store.Remove(id, p[0]+":"+p[2])
	if err != nil {
		log.Fatal(err)
	}
}
func (node *Node) Init(n *NodeConfig) {
	node.serfInit(n)
	node.raftInit(n)
	node.Queries = make(chan *serf.Query, 1000)
	node.UserEvents = make(chan *serf.UserEvent, 1000)
}

func (node *Node) Start() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Println(node.Raft().raft)
		case ev := <-node.Events():
			if memberEvent, ok := ev.(serf.MemberEvent); ok && node.Raft().Leader() {
				for _, member := range memberEvent.Members {
					if memberEvent.EventType() == serf.EventMemberJoin {
						node.JoinRaft(member.Name)
					} else if memberEvent.EventType() == serf.EventMemberLeave || memberEvent.EventType() == serf.EventMemberFailed || memberEvent.EventType() == serf.EventMemberReap {
						node.LeaveRaft(member.Name)
					}
				}
			} else if ev.EventType() == serf.EventUser {
				node.UserEvents <- ev.(*serf.UserEvent)
			} else if ev.EventType() == serf.EventQuery {
				fmt.Println("GOT EVENT:", ev)
				node.Queries <- ev.(*serf.Query)
			}
		}
	}
}
