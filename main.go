package main

import (
	"github.com/sophron-dev-works/legion/dbshell"
	"github.com/sophron-dev-works/legion/execution"
	"github.com/sophron-dev-works/legion/ui"
)

func startUI() {
	iw := ui.Init()
	var ee execution.ExecutionEngine
	var se dbshell.ShellEngine

	iw.AddApp("/scripts", &ee)
	iw.AddApp("/shell", &se)
	iw.Start()
}
func main() {
	startUI()

	// t := make([]byte, 0, 10000)
	// start := time.Now()
	// t = msgp.AppendArrayHeader(t, 100000*3)
	// for i := 0; i < 100000; i++ {
	// 	t = msgp.AppendArrayHeader(t, 1)
	// 	t = msgp.AppendInt(t, 10)
	// 	t = msgp.AppendArrayHeader(t, 1)
	// 	t = msgp.AppendString(t, "Hello")
	// 	t = msgp.AppendArrayHeader(t, 1)
	// 	t = msgp.AppendFloat64(t, 1.15)
	// }
	// var sout, sout2 bytes.Buffer
	// fmt.Println(msgp.ArrayHeaderSize)
	// fmt.Println(sout.Len(), time.Since(start))
	// msgp.UnmarshalAsJSON(&sout, t)

	// tt := make([]byte, 0, 10000)
	// start = time.Now()
	// for i := 0; i < 100000; i++ {
	// 	tt = msgp.AppendArrayHeader(tt, 1)
	// 	tt = msgp.AppendInt(tt, 10)
	// 	tt = msgp.AppendArrayHeader(tt, 1)
	// 	tt = msgp.AppendString(tt, "Hello")
	// 	tt = msgp.AppendArrayHeader(tt, 1)
	// 	tt = msgp.AppendFloat64(tt, 1.15)
	// }
	// tt = append(msgp.AppendArrayHeader([]byte{}, 100000*3), tt...)
	// fmt.Println(sout2.Len(), time.Since(start))
	// msgp.UnmarshalAsJSON(&sout2, tt)
	// fmt.Println(sout2.Len(), time.Since(start))

	// var x storage.SimpleStore
	// defer x.Shutdown()
	// x.Init("basic")
	// tableName := "datatypes_test"

	// s := time.Now()
	// _, err := x.Query("CREATE TABLE " + tableName + "(foo INTEGER,bar DOUBLE,bl BOOL,there STRING)")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(time.Since(s))
	// _, err = x.Query("INSERT INTO " + tableName + "(foo,bar,bl,there) VALUES ('4','4.5','true','hello'),('5','5.5','false','world')")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(time.Since(s))
	// xr, err := x.Query("SELECT foo,bar,bl,there from " + tableName)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// s = time.Now()
	// var b bytes.Buffer
	// msgp.Encode(&b, xr)
	// fmt.Println(time.Since(s))
	// x.Query("SELECT bar from " + tableName)
	// x.Query("SELECT foo from " + tableName)
}
