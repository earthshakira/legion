package dbshell

import (
	"bytes"
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/sophron-dev-works/legion/clustering"
	"github.com/sophron-dev-works/legion/storage"
	"github.com/sophron-dev-works/legion/ui"
	"github.com/tinylib/msgp/msgp"
)

// DistributedShell is used to display the db query shell on the browser and will query and return the data
type DistributedShell struct {
	vs   ui.ViewState
	db   storage.SimpleStore
	node *clustering.Node
}

func (se *DistributedShell) defaultConfig() map[string]interface{} {

	return map[string]interface{}{
		"vs": se.vs,
	}
}

func (se *DistributedShell) Init(db storage.SimpleStore, node *clustering.Node) {
	se.node = node
	se.db = db
}

// Create is the implementation of IrisApp for getting the party for the DB Shell
func (se *DistributedShell) Create(iw router.Party, vs ui.ViewState) {
	se.vs = vs
	se.vs.ActivePage = "Distributed Shell"
	iw.Get("/", func(ctx iris.Context) {
		ctx.View("dshell.html", se.defaultConfig())
	})
	iw.Get("/schemas", func(ctx iris.Context) {
		ctx.JSON(se.db.GetSchemas())
	})
	iw.Post("/query", se.executeQuery)
}

func (se *DistributedShell) Clean() {
	se.db.Shutdown()
}

func (se *DistributedShell) Listen() {
	for q := range se.node.Queries {
		query := string(q.Payload)
		qr, _ := se.db.Query(query)
		var b bytes.Buffer
		msgp.Encode(&b, qr)
		q.Respond(b.Bytes())
	}
}
func (se *DistributedShell) executeQuery(ctx iris.Context) {
	b := make(map[string]string)
	err := ctx.ReadJSON(&b)
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to parse payload").DetailErr(err))
		return
	}
	resp, _ := se.node.Serf().Query("QUERY", []byte(b["query"]), nil)
	ch := resp.ResponseCh()
	totalRows := uint32(0)
	var n uint32
	fields := []string{}
	field := ""
	payload := make([]byte, 0, 1000)
	for c := range ch {
		fields = []string{}
		_, c.Payload, err = msgp.ReadArrayHeaderBytes(c.Payload)
		n, c.Payload, err = msgp.ReadArrayHeaderBytes(c.Payload)
		for i := uint32(0); i < n; i++ {
			field, c.Payload, err = msgp.ReadStringBytes(c.Payload)
			fields = append(fields, field)
		}
		n, c.Payload, err = msgp.ReadArrayHeaderBytes(c.Payload)
		totalRows += n
		payload = append(payload, c.Payload...)
	}
	n = uint32(len(fields))
	response := make([]byte, 0, 1000)
	response = msgp.AppendArrayHeader(response, 2)
	response = msgp.AppendArrayHeader(response, n)
	for _, f := range fields {
		response = msgp.AppendString(response, f)
	}
	response = msgp.AppendArrayHeader(response, totalRows)
	response = append(response, payload...)
	fmt.Println("done with query Processing")
	resp.Close()
	// if qr != nil {
	// 	var b bytes.Buffer
	// 	msgp.Encode(&b, qr)

	msgp.UnmarshalAsJSON(ctx.ResponseWriter(), response)
	// 	return
	// }

	// if err != nil {
	// 	b["error"] = err.Error()
	// } else {
	// 	b["status"] = "query executed successfully"
	// }
	// ctx.JSON(b)
}
