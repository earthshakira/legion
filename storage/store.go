package storage

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/sophron-dev-works/legion/clustering"

	"github.com/dgraph-io/badger"
	"github.com/sophron-dev-works/legion/sqlparser"
	"github.com/sophron-dev-works/legion/sqlparser/query"
	"github.com/tinylib/msgp/msgp"
)

type SimpleStore struct {
	data          *badger.DB
	node          *clustering.Node
	autoIncrement map[string]*badger.Sequence
}

func (ss *SimpleStore) Init(node *clustering.Node) {
	var err error
	ss.autoIncrement = make(map[string]*badger.Sequence)
	ss.node = node

	if err != nil {
		log.Fatal(err)
	}
	ss.data, err = badger.Open(badger.DefaultOptions("/tmp/data_" + node.Id))
}

func (ss *SimpleStore) Shutdown() {
	for _, seq := range ss.autoIncrement {
		seq.Release()
	}
	ss.data.Close()
}

func (ss *SimpleStore) GetSchemas() []Schema {
	var s []Schema
	m, _ := ss.node.Raft().GetAll("table.")
	for _, v := range m {
		var ss Schema
		msgp.Decode(bytes.NewReader(v), &ss)
		s = append(s, ss)
	}
	return s
}

func (ss *SimpleStore) getAutoIncrement(schema Schema) (*badger.Sequence, error) {
	var err error
	if _, exists := ss.autoIncrement[schema.TableName]; !exists {
		ss.autoIncrement[schema.TableName], err = ss.data.GetSequence([]byte("auto_increment."+schema.TableName), 1000)
	}
	return ss.autoIncrement[schema.TableName], err
}

func (ss *SimpleStore) create(q query.Query) error {
	if _, exists := ss.node.Raft().Get(q.TableName); exists {
		return errors.New("Table '" + q.TableName + "' already exists")
	}
	var schema Schema
	schema.Init(q.TableName, q.Fields, q.DTypes)
	var b bytes.Buffer
	msgp.Encode(&b, &schema)
	err := ss.node.Raft().Set("table."+schema.TableName, b.Bytes())
	return err
}

func (ss *SimpleStore) getSchema(tableName string) (Schema, bool) {
	var s Schema
	v, exists := ss.node.Raft().Get("table." + tableName)
	if exists {
		msgp.Decode(bytes.NewReader(v), &s)
	}
	return s, exists
}
func (ss *SimpleStore) insert(q query.Query) error {
	schema, ok := ss.getSchema(q.TableName)
	if !ok {
		return errors.New(`No table with name '` + q.TableName + `'`)
	}
	err := schema.ValidateFields(q.Fields)
	if err != nil {
		return err
	}
	entries := make(chan Entry, 5)
	var wg sync.WaitGroup
	// TODO: test if this go routine helps after GEO2D
	go schema.Write(q.Fields, q.Inserts, entries, &wg)
	seq, err := ss.getAutoIncrement(schema)
	if err != nil {
		return err
	}
	wb := ss.data.NewWriteBatch()
	defer wb.Cancel()
	for c := range entries {
		if c.Err != nil {
			return c.Err
		}
		seqNum, err := seq.Next()
		if err != nil {
			return err
		}
		err = wb.Set([]byte(fmt.Sprintf("%s.%d", schema.TableName, seqNum)), c.Value)
		if err != nil {
			return err
		}
		wg.Done()
	}
	err = wb.Flush()
	return err
}

func (ss *SimpleStore) qselect(q query.Query) (*QueryResult, error) {
	schema, ok := ss.getSchema(q.TableName)
	if !ok {
		return nil, errors.New("No table with name '" + q.TableName + "'")
	}
	q.Fields = schema.UpdateStar(q.Fields)
	err := schema.ValidateFields(q.Fields)
	if err != nil {
		return nil, err
	}
	var qr QueryResult
	qr.Init(schema, q.Fields)
	err = ss.data.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(schema.TableName + ".")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(v []byte) error {
				qr.Project(v)
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &qr, nil
}

func (ss *SimpleStore) Query(queryStr string) (*QueryResult, error) {
	queryStr = strings.ReplaceAll(queryStr, "\n", "")
	q, err := sqlparser.Parse(queryStr)
	var res *QueryResult
	switch q.Type {
	case query.Create:
		err = ss.create(q)
	case query.Select:
		res, err = ss.qselect(q)
	case query.Insert:
		err = ss.insert(q)
	case query.Update:
		fmt.Println("Update query")
	case query.Delete:
		fmt.Println("Delete query")
	default:
		fmt.Println("Unknown")
	}
	return res, err
}
