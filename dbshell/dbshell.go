package dbshell

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/sophron-dev-works/legion/ui"
)

// ShellEngine is used to display the db query shell on the browser and will query and return the data
type ShellEngine struct {
	vs ui.ViewState
}

func (se *ShellEngine) defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"vs": se.vs,
	}
}

// Create is the implementation of IrisApp for getting the party for the DB Shell
func (se *ShellEngine) Create(iw router.Party, vs ui.ViewState) {
	se.vs = vs
	iw.Get("/", func(ctx iris.Context) {
		ctx.View("dbshell.html", se.defaultConfig())
	})
}
