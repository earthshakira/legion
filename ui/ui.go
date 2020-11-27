package ui

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/mvc"
	"google.golang.org/grpc"
)

// Breadcrumb TODO: Breadcrumb documentation
type Breadcrumb struct {
	Link string
	Name string
}

// CardData TODO: CardData documentation
type CardData struct {
	Label string
	Title string
}

// NavItem TODO: NavItem documentation
type NavItem struct {
	Type string
	Name string
	Link string
	Icon string
}

// ViewState TODO: ViewState documentation
type ViewState struct {
	PageTitle   string
	Breadcrumbs []Breadcrumb
	ActivePage  string
	NavItems    []NavItem
}

var navItems []NavItem

// IrisWrapper hels us to create the default struct that holds the top level of the routes and does stuff like template initialization and other things
type IrisWrapper struct {
	app     *iris.Application
	engines []IrisApp
	vs      ViewState
	Grpc    *grpc.Server
}

// Init initializes the IrisWrapper with configs and instantiates the app
// TODO: use configs to help with routes
func Init() IrisWrapper {
	navItems = []NavItem{
		{
			"Link",
			"Dashboard",
			"/dashboard",
			"ni ni-app text-primary",
		},
		{
			"Heading",
			"Execution",
			"",
			"",
		},
		{
			"Link",
			"Scripts",
			"/scripts",
			"ni ni-collection text-orange",
		},
		{
			"Link",
			"Workflow",
			"/workflows",
			"ni ni-vector text-purple",
		},
		{
			"Heading",
			"Database",
			"",
			"",
		},
		{
			"Link",
			"DB Shell",
			"/shell",
			"ni ni-laptop text-blue",
		},
		{
			"Link",
			"Distributed Shell",
			"/dshell",
			"ni ni-cloud-download-95 text-green",
		},
	}
	var iw IrisWrapper
	iw.app = iris.New()
	iw.app.HandleDir("/files", iris.Dir("/tmp/analysis"), iris.DirOptions{
		Compress: true,
		ShowList: true,
	})
	iw.Grpc = grpc.NewServer()
	iw.vs = ViewState{
		ActivePage: "Scripts",
		NavItems:   navItems,
		// Cards: []CardData{
		// 	{"New Scripts", "Create a new Script"},
		// 	{"Scripts", "Edit Scripts"},
		// },
		// Script: scriptPayload,
	}
	return iw
}

// Start : member function to initialize the wrapper
func (iw *IrisWrapper) Start(httpPort int) {
	iw.app.HandleDir("/assets", iris.Dir("ui/assets"))
	iw.app.RegisterView(iris.Django("ui/views", ".html").Reload(true))
	iw.app.Listen(fmt.Sprintf("0.0.0.0:%d", httpPort))
}

// IrisApp is the interface that allows an app to create more routes
type IrisApp interface {
	Create(iw router.Party, vs ViewState)
	Clean()
}

// AddApp uses the IrisApp interface and executes the `Create` method to add the routes defined by the related app
func (iw *IrisWrapper) AddApp(route string, ia IrisApp) {
	iw.engines = append(iw.engines, ia)
	party := iw.app.Party(route)
	ia.Create(party, iw.vs)
}

func (iw *IrisWrapper) RegistergRPC(service interface{}) {
	rootApp := mvc.New(iw.app)
	rootApp.Handle(&service, mvc.GRPC{
		Server:      iw.Grpc,          // Required.
		ServiceName: "proto.Executor", // Required.
		Strict:      false,
	})
}

func (iw *IrisWrapper) Clean() {
	for _, engine := range iw.engines {
		engine.Clean()
	}
}
