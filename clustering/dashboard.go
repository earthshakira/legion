package clustering

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/serf/serf"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/sophron-dev-works/legion/ui"
)

type Dashboard struct {
	vs        ui.ViewState
	node      *Node
	Workflows int
	Tasks     int
}

func (dash *Dashboard) Init(node *Node) {
	dash.node = node
}

func (dash *Dashboard) defaultConfig() map[string]interface{} {
	dash.vs.ActivePage = "Dashboard"
	s := dash.node.Serf()

	// GetCoordinate, err := s.GetCoordinate() // (*coordinate.Coordinate, error)
	member := s.LocalMember()    // Member
	memberList := s.Memberlist() // []Member
	members := s.Members()       // []Member
	numNodes := s.NumNodes()     // (numNodes int)
	State := s.State().String()  // SerfState
	stats := s.Stats()           // map[string]string
	healthyNodes := 0
	for _, m := range members {
		if m.Status == serf.StatusAlive {
			healthyNodes += 1
		}
	}
	memory, err := memory.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	gb := 1024 * 1024 * 1024 * 1.0
	before, err := cpu.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	time.Sleep(time.Duration(1) * time.Second)
	after, err := cpu.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	total := float64(after.Total - before.Total)
	return map[string]interface{}{
		"vs":           dash.vs,
		"memberList":   memberList,
		"member":       member,
		"members":      members,
		"numNodes":     numNodes,
		"healthyNodes": healthyNodes,
		"failedNodes":  numNodes - healthyNodes,
		"state":        State,
		"stat":         stats,
		"memorytotal":  fmt.Sprintf("%0.2f", float64(memory.Total)/gb),
		"memoryused":   fmt.Sprintf("%0.2f", float64(memory.Used)/gb),
		"memorycached": fmt.Sprintf("%0.2f", float64(memory.Cached)/gb),
		"memoryfree":   fmt.Sprintf("%0.2f", float64(memory.Free)/gb),
		"cpuuser":      fmt.Sprintf("%0.2f %%\n", float64(after.User-before.User)/total*100),
		"cpusystem":    fmt.Sprintf("%0.2f %%\n", float64(after.System-before.System)/total*100),
		"cpuidle":      fmt.Sprintf("%0.2f %%\n", float64(after.Idle-before.Idle)/total*100),
		"numkeys":      dash.node.store.GetSize(),
		"raftState":    dash.node.store.raft.State().String(),
	}
}

func (dash *Dashboard) Create(iw router.Party, vs ui.ViewState) {
	dash.vs = vs
	iw.Get("/", func(ctx iris.Context) {
		ctx.View("dashboard.html", dash.defaultConfig())
	})
}

func (dash *Dashboard) Clean() {
}
