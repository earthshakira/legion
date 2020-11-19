package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os/exec"

	"github.com/google/uuid"
	"github.com/sophron-dev-works/legion/clustering"
	"github.com/sophron-dev-works/legion/proto"
	"github.com/sophron-dev-works/legion/storage"
	"google.golang.org/grpc"
)

type Executor struct {
	node         *clustering.Node
	tasks        chan proto.Task
	runningTasks map[string]proto.Task
	workFlowlogs map[string]string
	socket       string
	db           *storage.SimpleStore
}

func (ex *Executor) Init(node *clustering.Node, nc *clustering.NodeConfig, db *storage.SimpleStore) {
	ex.db = db
	ex.socket = fmt.Sprintf("%s:%d", nc.Ip, nc.GrpcPort)
	ex.node = node
	ex.tasks = make(chan proto.Task, 100)
	ex.runningTasks = make(map[string]proto.Task)
	ex.workFlowlogs = make(map[string]string)
}

func (ex *Executor) Submit(c context.Context, p *proto.Task) (*proto.TaskAck, error) {
	ex.tasks <- *p
	return &proto.TaskAck{}, nil
}

func (ex *Executor) WriteFile(ctx context.Context, in *proto.FileContents) (*proto.FilePath, error) {
	file, err := ioutil.TempFile("/tmp", "workflow_*")
	abspath := file.Name()
	defer file.Close()
	if err != nil {
		return nil, err
	}
	file.WriteString(in.Content)
	return &proto.FilePath{Path: abspath}, nil
}

func (ex *Executor) execWorkflow(wp WorkflowPayload) (string, error) {
	task := proto.Task{
		Id:              uuid.New().String(),
		WorkflowId:      uuid.New().String(),
		WorkflowName:    wp.Name,
		GraphIndex:      0,
		OwnerRpcAddress: ex.node.GRPCEndpoint,
	}
	ex.tasks <- task
	ex.workFlowlogs[task.GetWorkflowId()] = "Workflow created"
	return task.GetWorkflowId(), nil
}

func (ex *Executor) getWorkflow(wf string) (*WorkflowPayload, error) {
	val, exists := ex.node.Raft().Get("workflows." + wf)
	if !exists {
		return nil, errors.New("No such Workflow")
	}
	var wp WorkflowPayload
	err := wp.FromBytes(val)
	return &wp, err
}
func (ex *Executor) execScript(block WorkflowBlock, task proto.Task) (string, error) {
	fmt.Println("Exec Script", block.Value)
	flags := []string{}
	if inpFile := task.GetInputFile(); inpFile != "" {
		flags = append(flags, "--input")
		flags = append(flags, inpFile)
	}

	file, err := ioutil.TempFile("/tmp", "workflow_*")
	outputPath := file.Name()
	file.Close()
	flags = append(flags, "--output")
	flags = append(flags, outputPath)

	file, err = ioutil.TempFile("/tmp", "script_*.py")
	if err != nil {
		fmt.Println("ERR", err)
	}
	abspath := file.Name()

	var sp ScriptPayload
	val, exists := ex.node.Raft().Get("script." + block.Value)
	if !exists {
		fmt.Println("ERR: script doesnt Exist")
	}
	err = sp.FromBytes(val)

	err = ioutil.WriteFile(abspath, []byte(sp.Text), 0644)
	goExecutable, _ := exec.LookPath("python3")
	var sout, serr bytes.Buffer
	cmdGoVer := &exec.Cmd{
		Path:   goExecutable,
		Args:   append([]string{goExecutable, abspath}, flags...),
		Stdout: &sout,
		Stderr: &serr,
	}
	fmt.Println(cmdGoVer.String())
	err = cmdGoVer.Run()
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	fmt.Println("serr:", serr.String())
	fmt.Println("sout:", sout.String())
	return outputPath, err
}

func getNextTasks(wf *WorkflowPayload, node int, inp string, parentTask proto.Task) []*proto.Task {
	tasks := []*proto.Task{}
	for idx, block := range wf.Graph {
		if block.Parent == node {
			task := proto.Task{
				Id:              uuid.New().String(),
				WorkflowId:      parentTask.GetWorkflowId(),
				WorkflowName:    wf.Name,
				GraphIndex:      int32(idx),
				OwnerRpcAddress: parentTask.GetOwnerRpcAddress(),
				InputFile:       inp,
			}
			tasks = append(tasks, &task)
		}
	}
	return tasks
}

func (ex *Executor) JsonSplitDist(wf *WorkflowPayload, task proto.Task) {
	fmt.Println("Exec Json Split")
	nodeAddrs := []string{}
	for _, member := range ex.node.Serf().Members() {
		nodeAddrs = append(nodeAddrs, member.Tags["grpc"])
	}
	n := len(nodeAddrs)
	content, err := ioutil.ReadFile(task.GetInputFile())
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	var x []interface{}
	err = json.Unmarshal(content, &x)
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	nextNodes := []int{}
	for i, block := range wf.Graph {
		if block.Parent == int(task.GetGraphIndex()) {
			nextNodes = append(nextNodes, i)
		}
	}

	for _, nextNode := range nextNodes {
		for i, data := range x {
			p, err := json.Marshal(data)
			fmt.Println(i, string(p), err)
			conn, err := grpc.Dial(nodeAddrs[i%n], grpc.WithInsecure())
			if err != nil {
				fmt.Println("ERR in grpc for ", nodeAddrs[i%n])
			}
			client := proto.NewExecutorClient(conn)
			ctx := context.TODO()
			outfile, err := client.WriteFile(ctx, &proto.FileContents{
				Content: string(p),
			})
			ctx = context.TODO()
			newTask := &proto.Task{
				Id:              uuid.New().String(),
				WorkflowId:      task.GetWorkflowId(),
				WorkflowName:    wf.Name,
				GraphIndex:      int32(nextNode),
				OwnerRpcAddress: task.GetOwnerRpcAddress(),
				InputFile:       outfile.GetPath(),
			}
			resp, err := client.Submit(ctx, newTask)
			if err != nil {
				fmt.Println("ERROR:", err, resp)
			}
			conn.Close()
		}
	}
}

func (ex *Executor) InsertData(task proto.Task) {
	fmt.Println("exec insert Data")
	content, err := ioutil.ReadFile(task.GetInputFile())
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	q := string(content)
	_, err = ex.db.Query(q)
	fmt.Println("InsertData executed with error", err)
}
func (ex *Executor) Listen() {
	for task := range ex.tasks {
		fmt.Println("working on ", task)
		wf, err := ex.getWorkflow(task.GetWorkflowName())
		if err != nil {
			fmt.Println("[ERROR]: handle error in workflow processing", err)
			continue
		}
		block := wf.Graph[task.GetGraphIndex()]
		switch block.Type {
		case Script:
			outputFile, err := ex.execScript(block, task)
			if err != nil {
				fmt.Println(err)
				continue
			}
			next := getNextTasks(wf, int(task.GetGraphIndex()), outputFile, task)
			for _, x := range next {
				ex.tasks <- *x
			}
		case JSONSplitter:
			ex.JsonSplitDist(wf, task)
		case DatabaseInsert:
			ex.InsertData(task)
		}
	}
}
func (ex *Executor) Serve() {
	lis, err := net.Listen("tcp", ex.socket)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterExecutorServer(s, ex)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (ex *Executor) Close() {
	close(ex.tasks)
}
