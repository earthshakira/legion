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

type logTuple struct {
	wl *proto.WorkflowLog
	to string
}
type Executor struct {
	node         *clustering.Node
	tasks        chan proto.Task
	incomingLogs chan proto.WorkflowLog
	outgoingLogs chan *logTuple
	runningTasks map[string]proto.Task
	WorkFlowlogs map[string]*bytes.Buffer
	socket       string
	db           *storage.SimpleStore
	id           string
}

func (ex *Executor) Init(node *clustering.Node, nc *clustering.NodeConfig, db *storage.SimpleStore) {
	ex.id = nc.Id
	ex.db = db
	ex.socket = fmt.Sprintf("%s:%d", nc.Ip, nc.GrpcPort)
	ex.node = node
	ex.tasks = make(chan proto.Task, 100)
	ex.incomingLogs = make(chan proto.WorkflowLog, 1000)
	ex.outgoingLogs = make(chan *logTuple, 1000)
	ex.runningTasks = make(map[string]proto.Task)
	ex.WorkFlowlogs = make(map[string]*bytes.Buffer)

}

func (ex *Executor) Submit(c context.Context, p *proto.Task) (*proto.TaskAck, error) {
	ex.tasks <- *p
	return &proto.TaskAck{}, nil
}

func (ex *Executor) writeLog(task *proto.Task, contents string) {
	ex.outgoingLogs <- &logTuple{
		to: task.GetOwnerRpcAddress(),
		wl: &proto.WorkflowLog{
			Id:         ex.id,
			WorkflowId: task.GetWorkflowId(),
			TaskId:     task.GetId(),
			Log:        contents,
		},
	}
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
	ex.WorkFlowlogs[task.GetWorkflowId()] = &bytes.Buffer{}
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
func (ex *Executor) execScript(block WorkflowBlock, task *proto.Task) (string, error) {
	ex.writeLog(task, fmt.Sprintf("Executing Script %s", block.Value))
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
		ex.writeLog(task, fmt.Sprintf("[ERROR] %v", err))
	}
	abspath := file.Name()

	var sp ScriptPayload
	val, exists := ex.node.Raft().Get("script." + block.Value)
	if !exists {
		ex.writeLog(task, fmt.Sprintf("[ERROR]: Script Doesn't exist %s", block.Value))
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
	ex.writeLog(task, fmt.Sprintf("[EXEC COMMAND]: %s", cmdGoVer.String()))
	err = cmdGoVer.Run()
	if err != nil {
		ex.writeLog(task, fmt.Sprintf("[ERROR]: %v", err))
	}
	ex.writeLog(task, fmt.Sprintf("[STDERR]: %s", serr.String()))
	ex.writeLog(task, fmt.Sprintf("[STDOUT]: %s", sout.String()))
	return outputPath, err
}

func getNextTasks(wf *WorkflowPayload, node int, inp string, parentTask *proto.Task) []*proto.Task {
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

func (ex *Executor) JsonSplitDist(wf *WorkflowPayload, task *proto.Task) {
	ex.writeLog(task, "Splitting on JSON")
	nodeAddrs := []string{}
	for _, member := range ex.node.Serf().Members() {
		nodeAddrs = append(nodeAddrs, member.Tags["grpc"])
	}
	n := len(nodeAddrs)
	content, err := ioutil.ReadFile(task.GetInputFile())
	if err != nil {
		ex.writeLog(task, fmt.Sprintf("[ERROR] %v", err))
	}
	var x []interface{}
	err = json.Unmarshal(content, &x)
	if err != nil {
		ex.writeLog(task, fmt.Sprintf("[ERROR] %v", err))
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
				ex.writeLog(task, fmt.Sprintf("[ERROR] - %s - %v", nodeAddrs[i%n], err))
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
			_, err = client.Submit(ctx, newTask)
			if err != nil {
				ex.writeLog(task, fmt.Sprintf("[ERROR] - %s - %v", nodeAddrs[i%n], err))
			}
			conn.Close()
		}
	}
}

func (ex *Executor) InsertData(task *proto.Task) {
	ex.writeLog(task, fmt.Sprintf("InsertingData Starting"))
	content, err := ioutil.ReadFile(task.GetInputFile())
	if err != nil {
		ex.writeLog(task, fmt.Sprintf("[ERROR] %v", err))
	}
	q := string(content)
	_, err = ex.db.Query(q)
	if err != nil {
		ex.writeLog(task, fmt.Sprintf("[ERROR] %v", err))
	} else {
		ex.writeLog(task, fmt.Sprintf("Data Insert Completed Successfully"))
	}
}

func (ex *Executor) Listen() {
	go func() {
		for log := range ex.incomingLogs {
			buf := ex.WorkFlowlogs[log.GetWorkflowId()]
			buf.WriteString("[" + log.GetId() + "] ")
			buf.WriteString("[" + log.GetTaskId() + "] ")
			buf.WriteString(log.GetLog() + "\n")
		}
	}()

	go func() {
		for tup := range ex.outgoingLogs {
			conn, err := grpc.Dial(tup.to, grpc.WithInsecure())
			if err != nil {
				fmt.Println("ERR in grpc for ", tup.to, err)
			}
			client := proto.NewExecutorClient(conn)
			ctx := context.TODO()
			_, err = client.Log(ctx, tup.wl)
			if err != nil {
				fmt.Println("ERR in logging for ", tup.to, err)
			}
			conn.Close()
		}
	}()
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
			outputFile, err := ex.execScript(block, &task)
			if err != nil {
				fmt.Println(err)
				continue
			}
			next := getNextTasks(wf, int(task.GetGraphIndex()), outputFile, &task)
			for _, x := range next {
				ex.tasks <- *x
			}
		case JSONSplitter:
			ex.JsonSplitDist(wf, &task)
		case DatabaseInsert:
			ex.InsertData(&task)
		}
	}
}

func (ex *Executor) Log(ctx context.Context, in *proto.WorkflowLog) (*proto.WorkflowLogAck, error) {
	ex.incomingLogs <- *in
	return &proto.WorkflowLogAck{Ack: "yes"}, nil
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
