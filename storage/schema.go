package storage

import (
	"bytes"
	"errors"

	"sort"
	"sync"

	"github.com/sophron-dev-works/legion/sqlparser/query"
	"github.com/tinylib/msgp/msgp"
)

type Column struct {
	Name     string
	DataType query.DType
}

type Schema struct {
	Cols      []Column
	TableName string
	cmap      map[string]Column
}

type Entry struct {
	Err   error
	Value []byte
}

func (s *Schema) EncodeMsg(w *msgp.Writer) error {
	w.WriteString(s.TableName)

	n := len(s.Cols)
	w.WriteArrayHeader(uint32(n))
	for i := 0; i < n; i++ {
		w.WriteString(s.Cols[i].Name)
		w.WriteInt(int(s.Cols[i].DataType))
	}
	return nil
}

func (s *Schema) DecodeMsg(r *msgp.Reader) error {
	var err error
	s.TableName, err = r.ReadString()
	if err != nil {
		return err
	}
	n, err := r.ReadArrayHeader()
	s.Cols = make([]Column, n)
	s.cmap = make(map[string]Column)

	for i := uint32(0); i < n; i++ {
		s.Cols[i].Name, err = r.ReadString()
		if err != nil {
			return err
		}
		var h int
		h, err = r.ReadInt()
		s.Cols[i].DataType = query.DType(h)
		s.cmap[s.Cols[i].Name] = s.Cols[i]
		if err != nil {
			return err
		}
	}
	return nil
}

type QueryResult struct {
	Cols     []Column
	RowCount uint32
	Result   []byte
	pv       []bool
	s        Schema
	NumCols  uint32
}

func (qr *QueryResult) Init(s Schema, fields []string) {
	qr.RowCount = 0
	fmap := make(map[string]bool)
	qr.NumCols = uint32(len(fields))
	qr.s = s
	for _, f := range fields {
		fmap[f] = true
	}

	for _, c := range s.Cols {
		_, exists := fmap[c.Name]
		if exists {
			qr.Cols = append(qr.Cols, c)
		}
		qr.pv = append(qr.pv, exists)
	}

}

func (qr *QueryResult) Project(row []byte) {
	var prow []byte
	prow = msgp.AppendArrayHeader(prow, qr.NumCols)
	var err error
	for i, col := range qr.s.Cols {
		prow, err = ppack(&prow, &row, col.DataType, qr.pv[i])
		if err != nil {
			panic(err)
		}
	}

	var sout bytes.Buffer
	msgp.UnmarshalAsJSON(&sout, prow)
	qr.Result = append(qr.Result, prow...)
	qr.RowCount++
}

func (qr *QueryResult) EncodeMsg(w *msgp.Writer) error {
	w.WriteArrayHeader(2)
	w.WriteArrayHeader(uint32(len(qr.Cols)))
	for _, col := range qr.Cols {
		w.WriteString(col.Name)
	}
	w.WriteArrayHeader(qr.RowCount)
	w.Append(qr.Result...)
	return nil
}

func (s *Schema) Init(tableName string, fields []string, dtypes []query.DType) {
	s.TableName = tableName
	s.Cols = make([]Column, len(fields))
	s.cmap = make(map[string]Column)

	for i := 0; i < len(fields); i++ {
		s.Cols[i].Name = fields[i]
		s.Cols[i].DataType = dtypes[i]
	}

	sort.Slice(s.Cols, func(i, j int) bool {
		return s.Cols[i].Name < s.Cols[j].Name
	})

	for _, col := range s.Cols {
		s.cmap[col.Name] = col
	}

}
func (s *Schema) UpdateStar(fields []string) []string {
	if fields[0] == "*" {
		f := []string{}
		for _, col := range s.Cols {
			f = append(f, col.Name)
		}
		return f
	}
	return fields
}
func (s *Schema) ValidateFields(fields []string) error {
	for _, field := range fields {
		if _, exists := s.cmap[field]; !exists {
			return errors.New(s.TableName + " has no column " + field)
		}
	}
	return nil
}

func (s *Schema) empty(e *Entry, dt query.DType) {
	e.Value = msgp.AppendNil(e.Value)
}

func (s *Schema) getPositionMap(fields []string) map[string]int {
	positions := make(map[string]int)
	for i := 0; i < len(fields); i++ {
		positions[fields[i]] = i
	}
	return positions
}

func (s *Schema) Write(fields []string, values [][]string, entries chan Entry, wg *sync.WaitGroup) {
	defer func() {
		wg.Wait()
		close(entries)
	}()
	positions := s.getPositionMap(fields)
	for _, row := range values {
		wg.Add(1)
		var e Entry
		for _, col := range s.Cols {
			pos, exists := positions[col.Name]
			if exists {
				process(&e, row[pos], col.DataType)
			} else {
				s.empty(&e, col.DataType)
			}
		}
		entries <- e
		if e.Err != nil {
			return
		}
	}
}
