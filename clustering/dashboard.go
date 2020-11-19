package clustering

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/sophron-dev-works/legion/ui"
)

type Dashboard struct {
	vs   ui.ViewState
	node *Node
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
	stat := s.Stats()            // map[string]string
	// fmt.Println("memberList", memberList, "\n\n")
	// fmt.Println("member", member, "\n\n")
	// fmt.Println("members", members, "\n\n")
	// fmt.Println("numNodes", numNodes, "\n\n")
	// fmt.Println("State", State, "\n\n")
	// fmt.Println("stat", stat, "\n\n")
	return map[string]interface{}{
		"vs":         dash.vs,
		"memberList": memberList,
		"member":     member,
		"members":    members,
		"numNodes":   numNodes,
		"state":      State,
		"stat":       stat,
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
