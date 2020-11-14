package storage

import (
	"errors"
	"strconv"
	"time"

	"github.com/sophron-dev-works/legion/sqlparser/query"
	"github.com/tinylib/msgp/msgp"
)

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
		e.Err = errors.New("GEO2D: Not Yet Implemented")
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
		return *prow, errors.New("GEO2D not implemented")
	}
	return *prow, err
}
