package ui

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
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
	app *iris.Application
	vs  ViewState
}

// Init initializes the IrisWrapper with configs and instantiates the app
// TODO: use configs to help with routes
func Init() IrisWrapper {
	navItems = []NavItem{
		{
			"Link",
			"Dashboard",
			"",
			"ni ni-tv-2 text-primary",
		},
		{
			"Link",
			"Scripts",
			"/scripts",
			"ni ni-collection text-orange",
		},
		{
			"Link",
			"DB Shell",
			"/shell",
			"ni ni-laptop text-blue",
		},
		{
			"Heading",
			"Documentation",
			"",
			"",
		},
	}
	var iw IrisWrapper
	iw.app = iris.New()
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
func (iw *IrisWrapper) Start() {
	iw.app.HandleDir("/assets", iris.Dir("ui/assets"))
	iw.app.RegisterView(iris.Django("ui/views", ".html").Reload(true))
	iw.app.Listen(":8080")
}

// IrisApp is the interface that allows an app to create more routes
type IrisApp interface {
	Create(iw router.Party, vs ViewState)
}

// AddApp uses the IrisApp interface and executes the `Create` method to add the routes defined by the related app
func (iw *IrisWrapper) AddApp(route string, ia IrisApp) {
	party := iw.app.Party(route)
	ia.Create(party, iw.vs)
}
