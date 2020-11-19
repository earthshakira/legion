package storage

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"

	"github.com/sophron-dev-works/legion/sqlparser/query"
	"github.com/tinylib/msgp/msgp"
)

type Geo2D struct {
	Start     []float64   `json:"start"`
	End       []float64   `json:"end"`
	Precision float64     `json:"precision"`
	Data      [][]float64 `json:"data"`
}

func (g *Geo2D) EncodeMsg(w *msgp.Writer) error {
	w.WriteArrayHeader(4)

	w.WriteArrayHeader(2)
	w.WriteFloat64(g.Start[0])
	w.WriteFloat64(g.Start[1])

	w.WriteArrayHeader(2)
	w.WriteFloat64(g.End[0])
	w.WriteFloat64(g.End[1])

	w.WriteFloat64(g.Precision)
	rows := len(g.Data)
	cols := 0
	if rows > 0 {
		cols = len(g.Data[0])
	}
	w.WriteArrayHeader(2)

	w.WriteArrayHeader(2)
	w.WriteInt(rows)
	w.WriteInt(cols)

	w.WriteArrayHeader(uint32(rows * cols))
	for _, r := range g.Data {
		for _, d := range r {
			w.WriteFloat64(d)
		}
	}
	return nil
}

func (g *Geo2D) DecodeMsg(r *msgp.Reader) error {
	var err error
	g.Start = make([]float64, 2)
	g.End = make([]float64, 2)
	var rows, cols int
	_, _ = r.ReadArrayHeader()
	_, _ = r.ReadArrayHeader()
	g.Start[0], err = r.ReadFloat64()
	g.Start[1], err = r.ReadFloat64()
	_, _ = r.ReadArrayHeader()
	g.End[0], err = r.ReadFloat64()
	g.End[1], err = r.ReadFloat64()
	g.Precision, err = r.ReadFloat64()
	_, _ = r.ReadArrayHeader()
	_, _ = r.ReadArrayHeader()
	rows, err = r.ReadInt()
	cols, err = r.ReadInt()
	_, _ = r.ReadArrayHeader()
	g.Data = make([][]float64, rows)
	for i := 0; i < rows; i++ {
		g.Data[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			g.Data[i][j], err = r.ReadFloat64()
		}
	}
	return err
}
func process(e *Entry, value string, dt query.DType) {
	switch dt {
	case query.Int:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			e.Err = err
			return
		}
		e.Value = msgp.AppendInt64(e.Value, i)
	case query.Bool:
		i, err := strconv.ParseBool(value)
		if err != nil {
			e.Err = err
			return
		}
		e.Value = msgp.AppendBool(e.Value, i)
	case query.Double:
		i, err := strconv.ParseFloat(value, 64)
		if err != nil {
			e.Err = err
			return
		}
		e.Value = msgp.AppendFloat64(e.Value, i)
	case query.String:
		e.Value = msgp.AppendString(e.Value, value)
	case query.DateTime:
		i, err := time.Parse(time.RFC3339, value)
		if err != nil {
			e.Err = err
			return
		}
		e.Value = msgp.AppendInt64(e.Value, i.Unix())
	case query.Geo2d:
		var b bytes.Buffer
		var g Geo2D
		json.Unmarshal([]byte(value), &g)
		msgp.Encode(&b, &g)
		e.Value = msgp.AppendInt(e.Value, b.Len())
		e.Value = append(e.Value, b.Bytes()...)
	}
}

func ppack(prow *[]byte, row *[]byte, dt query.DType, mask bool) ([]byte, error) {
	var err error
	if msgp.IsNil(*row) {
		*prow = msgp.AppendNil(*prow)
		*row, err = msgp.ReadNilBytes(*row)
		return *prow, err
	}
	switch dt {
	case query.Int:
		var i int64
		i, *row, err = msgp.ReadInt64Bytes(*row)
		if mask {
			*prow = msgp.AppendInt64(*prow, i)
		}
	case query.Bool:
		var i bool
		i, *row, err = msgp.ReadBoolBytes(*row)
		if mask {
			*prow = msgp.AppendBool(*prow, i)
		}
	case query.Double:
		var i float64
		i, *row, err = msgp.ReadFloat64Bytes(*row)
		if mask {
			*prow = msgp.AppendFloat64(*prow, i)
		}
	case query.String:
		var i string
		i, *row, err = msgp.ReadStringBytes(*row)
		if mask {
			*prow = msgp.AppendString(*prow, i)
		}
	case query.DateTime:
		var i int64
		i, *row, err = msgp.ReadInt64Bytes(*row)
		if mask {
			*prow = msgp.AppendInt64(*prow, i)
		}
	case query.Geo2d:
		var l int
		l, *row, err = msgp.ReadIntBytes(*row)
		if mask {
			*prow = append(*prow, (*row)[:l]...)
		}
		*row = (*row)[l:]
	}
	return *prow, err
}
