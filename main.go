package main

import (
	"github.com/sophron-dev-works/legion/dbshell"
	"github.com/sophron-dev-works/legion/execution"
	"github.com/sophron-dev-works/legion/sqlparser"
	"github.com/sophron-dev-works/legion/ui"
)

func main() {
	iw := ui.Init()
	var ee execution.ExecutionEngine
	var se dbshell.ShellEngine

	iw.AddApp("/scripts", &ee)
	iw.AddApp("/shell", &se)
	iw.Start()
	// scanner := bufio.NewScanner(os.Stdin)
	// var queryText string
	// for scanner.Scan() {
	// 	queryText = scanner.Text()
	// 	break
	// }
	sqlparser.ParserTests()
	// q, _ := sqlparser.Parse("CREATE TABLE shubham(x INTEGER,y STRING,z BOOL)")
	// switch q.Type {
	// case query.Select:
	// 	fmt.Println("Select query")
	// case query.Update:
	// 	fmt.Println("Update query")
	// case query.Insert:
	// 	fmt.Println("Insert query")
	// case query.Delete:
	// 	fmt.Println("Delete query")
	// case query.Create:
	// 	fmt.Println("Create query")
	// default:
	// 	fmt.Println("Unknown")
	// }
	// fmt.Println(q)
}
