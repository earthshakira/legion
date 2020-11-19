package execution

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/sophron-dev-works/legion/clustering"
	"github.com/sophron-dev-works/legion/ui"
)

type WorkflowEngine struct {
	vs       ui.ViewState
	node     *clustering.Node
	executor Executor
}

func (we *WorkflowEngine) Init(node *clustering.Node, executor Executor) {
	we.node = node
	we.executor = executor
}

func (we *WorkflowEngine) defaultConfig() map[string]interface{} {
	we.vs.ActivePage = "Workflows"
	w, _ := we.node.Raft().GetAll("workflows.")
	workflows := []WorkflowPayload{}
	for _, v := range w {
		var wf WorkflowPayload
		wf.FromBytes(v)
		workflows = append(workflows, wf)
	}

	s, _ := we.node.Raft().GetAll("script.")
	scripts := []ScriptPayload{}
	for _, v := range s {
		var sp ScriptPayload
		sp.FromBytes(v)
		scripts = append(scripts, sp)
	}
	print(scripts)
	return map[string]interface{}{
		"vs":             we.vs,
		"savedWorkflows": workflows,
		"savedScripts":   scripts,
	}
}

func (we *WorkflowEngine) Create(iw router.Party, vs ui.ViewState) {
	we.vs = vs
	iw.Get("/", func(ctx iris.Context) {
		ctx.View("workflows.html", we.defaultConfig())
	})

	iw.Get("/view/{name}", func(ctx iris.Context) {
		// TODO: handle errors
		workflowPayload, _ := we.getWorkflow(ctx.Params().Get("name"))
		data := we.defaultConfig()
		data["workflow"] = workflowPayload
		output, _ := json.Marshal(workflowPayload.Output)
		data["workflowOutput"] = string(output)
		ctx.View("workflows.html", data)
	})
	iw.Get("/list", we.listAllWorkflows)
	iw.Get("/execute/{name}", we.executeWorkflow)
	iw.Post("/save", we.saveWorkflow)
}

func (we *WorkflowEngine) Clean() {
}

func (we *WorkflowEngine) getWorkflow(name string) (WorkflowPayload, error) {
	var wp WorkflowPayload
	val, exists := we.node.Raft().Get("workflows." + name)
	if !exists {
		return wp, errors.New("Script doesn't exist")
	}
	err := wp.FromBytes(val)
	return wp, err
}

func (we *WorkflowEngine) listAllWorkflows(ctx iris.Context) {
	var workflows []WorkflowPayload
	m, _ := we.node.Raft().GetAll("workflows.")
	for _, v := range m {
		var sp WorkflowPayload
		sp.FromBytes(v)
		workflows = append(workflows, sp)
	}
	ctx.JSON(workflows)
}
func (we *WorkflowEngine) saveWorkflow(ctx iris.Context) {
	var b WorkflowPayload
	err := ctx.ReadJSON(&b)
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to parse payload").DetailErr(err))
		return
	}
	b.Modified = time.Now()
	err = we.node.Raft().Set("workflows."+b.Name, b.Bytes())
	if err != nil {
		ctx.StopWithProblem(iris.StatusForbidden, iris.NewProblem().
			Title("Unable to save workflow").DetailErr(err))
	}
	ctx.JSON(&b)
	ctx.StatusCode(iris.StatusCreated)
}

// func (we *WorkflowEngine) runScript(name string, flags string) (*ScriptOutput, error) {
// 	dir, err := ioutil.TempDir("/tmp", "script")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer os.RemoveAll(dir)
// 	fmt.Println(dir)
// 	fileName := dir + "/script.py"
// 	err = ioutil.WriteFile(fileName, []byte(b.Text), 0644)

// 	goExecutable, _ := exec.LookPath("python3")
// 	var sout, serr bytes.Buffer
// 	cmdGoVer := &exec.Cmd{
// 		Path:   goExecutable,
// 		Args:   []string{goExecutable, file},
// 		Stdout: &sout,
// 		Stderr: &serr,
// 	}
// 	err := cmdGoVer.Run()
// 	if err != nil {
// 		fmt.Println("ERROR:", err)
// 	}
// 	fmt.Println(serr.String())
// 	return &ScriptOutput{
// 		Cmd:    cmdGoVer.String(),
// 		Stderr: sout.String(),
// 		Stdout: serr.String(),
// 	}, nil
// }
func (we *WorkflowEngine) executeWorkflow(ctx iris.Context) {
	workflowPayload, err := we.getWorkflow(ctx.Params().Get("name"))
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Unable to execute script").DetailErr(err))
		return
	}
	tracker := make(map[string]string)
	wid, err := we.executor.execWorkflow(workflowPayload)
	if err != nil {
		tracker["err"] = err.Error()
	} else {
		tracker["workflowId"] = wid
		ctx.StatusCode(iris.StatusCreated)
	}
	ctx.JSON(tracker)
}
