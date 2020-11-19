package storage

import (
	"fmt"
	"time"
)

func doBenchmarks() {
	var x SimpleStore
	defer x.Shutdown()
	x.Init(nil)
	tableName := "datatypes_test"

	s := time.Now()
	_, err := x.Query("CREATE TABLE " + tableName + "(foo INTEGER,bar DOUBLE,bl BOOL,there STRING)")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(time.Since(s))
	s = time.Now()
	q := "INSERT INTO " + tableName + "(foo,bar,bl,there) VALUES ('4','4.5','true','hello'),('5','5.5','false','world')"
	for i := 0; i < 1000; i++ {
		_, _ = x.Query(q)
	}
	fmt.Println("1000 inserts", time.Since(s))
	s = time.Now()
	xr, err := x.Query("SELECT foo,bar,bl,there from " + tableName)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(xr.RowCount)
	}

	// s = time.Now()
	// var b bytes.Buffer
	// msgp.Encode(&b, xr)
	fmt.Println(time.Since(s))
	// x.Query("SELECT bar from " + tableName)
	// x.Query("SELECT foo from " + tableName)
}
