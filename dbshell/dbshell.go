package dbshell

import (
	"bytes"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/sophron-dev-works/legion/clustering"
	"github.com/sophron-dev-works/legion/storage"
	"github.com/sophron-dev-works/legion/ui"
	"github.com/tinylib/msgp/msgp"
)

// ShellEngine is used to display the db query shell on the browser and will query and return the data
type ShellEngine struct {
	vs ui.ViewState
	db storage.SimpleStore
}

func (se *ShellEngine) defaultConfig() map[string]interface{} {

	return map[string]interface{}{
		"vs": se.vs,
	}
}

func (se *ShellEngine) SetNode(db storage.SimpleStore, node *clustering.Node) {
	se.db = db
}

// Create is the implementation of IrisApp for getting the party for the DB Shell
func (se *ShellEngine) Create(iw router.Party, vs ui.ViewState) {
	se.vs = vs
	se.vs.ActivePage = "DB Shell"
	iw.Get("/", func(ctx iris.Context) {
		ctx.View("dbshell.html", se.defaultConfig())
	})
	iw.Get("/schemas", func(ctx iris.Context) {
		ctx.JSON(se.db.GetSchemas())
	})
	iw.Post("/query", se.executeQuery)
}

func (se *ShellEngine) Clean() {
	se.db.Shutdown()
}
func (se *ShellEngine) executeQuery(ctx iris.Context) {
	b := make(map[string]string)
	err := ctx.ReadJSON(&b)
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to parse payload").DetailErr(err))
		return
	}
	qr, err := se.db.Query(b["query"])
	if qr != nil {
		var b bytes.Buffer
		msgp.Encode(&b, qr)
		msgp.UnmarshalAsJSON(ctx.ResponseWriter(), b.Bytes())
		return
	}

	if err != nil {
		b["error"] = err.Error()
	} else {
		b["status"] = "query executed successfully"
	}
	ctx.JSON(b)
}
