package execution

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/sophron-dev-works/legion/clustering"
	"github.com/sophron-dev-works/legion/ui"
)

func RunScript(file string, wg *sync.WaitGroup) (string, string, string) {
	wg.Add(1)
	goExecutable, _ := exec.LookPath("python3")
	var sout, serr bytes.Buffer
	cmdGoVer := &exec.Cmd{
		Path:   goExecutable,
		Args:   []string{goExecutable, file},
		Stdout: &sout,
		Stderr: &serr,
	}
	err := cmdGoVer.Run()
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	fmt.Println(serr.String())
	wg.Done()
	return cmdGoVer.String(), string(sout.String()), string(serr.String())
}

type ExecutionEngine struct {
	vs   ui.ViewState
	node *clustering.Node
}

func (ee *ExecutionEngine) Init(node *clustering.Node) {
	ee.node = node
}

func (ee *ExecutionEngine) defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"vs": ee.vs,
	}
}

func (ee *ExecutionEngine) Create(iw router.Party, vs ui.ViewState) {
	ee.vs = vs
	iw.Get("/", func(ctx iris.Context) {
		ctx.View("execution.html", ee.defaultConfig())
	})

	iw.Get("/view/{name}", func(ctx iris.Context) {
		// TODO: handle errors
		scriptPayload, _ := ee.getScript(ctx.Params().Get("name"))
		data := ee.defaultConfig()
		data["script"] = scriptPayload
		ctx.View("execution.html", data)
	})
	iw.Get("/list", ee.listAllScripts)
	iw.Post("/execute", ee.executeScript)
	iw.Post("/save", ee.saveScript)
}

func (ee *ExecutionEngine) Clean() {

}
func (ee *ExecutionEngine) getScript(name string) (ScriptPayload, error) {
	var sp ScriptPayload
	val, exists := ee.node.Raft().Get("script." + name)
	if !exists {
		return sp, errors.New("Script doesn't exist")
	}
	err := sp.FromBytes(val)
	return sp, err
}

func (ee *ExecutionEngine) listAllScripts(ctx iris.Context) {
	var scripts []ScriptPayload
	m, _ := ee.node.Raft().GetAll("script.")
	for _, v := range m {
		var sp ScriptPayload
		sp.FromBytes(v)
		scripts = append(scripts, sp)
	}
	ctx.JSON(scripts)
}
func (ee *ExecutionEngine) saveScript(ctx iris.Context) {
	var b ScriptPayload
	err := ctx.ReadJSON(&b)
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to parse payload").DetailErr(err))
		return
	}

	b.Modified = time.Now()
	err = ee.node.Raft().Set("script."+b.Name, b.Bytes())
	if err != nil {
		ctx.StopWithProblem(iris.StatusForbidden, iris.NewProblem().
			Title("Unable to save script").DetailErr(err))
	}
	ctx.StatusCode(iris.StatusCreated)
}

func (ee *ExecutionEngine) executeScript(ctx iris.Context) {
	var b ScriptPayload
	err := ctx.ReadJSON(&b)

	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to parse payload").DetailErr(err))
		return
	}

	dir, err := ioutil.TempDir("/tmp", "script")
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to execute script").DetailErr(err))
		return
	}
	defer os.RemoveAll(dir)
	fmt.Println(dir)
	fileName := dir + "/script.py"
	err = ioutil.WriteFile(fileName, []byte(b.Text), 0644)
	out, err := exec.Command("python3", fileName).Output()
	fmt.Println(out)
	var wg sync.WaitGroup
	cmd, op, er := RunScript(fileName, &wg)
	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(ScriptOutput{cmd, er, op})
}
