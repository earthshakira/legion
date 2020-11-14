package storage

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/dgraph-io/badger"
	"github.com/sophron-dev-works/legion/sqlparser"
	"github.com/sophron-dev-works/legion/sqlparser/query"
	"github.com/tinylib/msgp/msgp"
)

type SimpleStore struct {
	data     *badger.DB
	metadata *badger.DB
	schema   map[string]Schema
}

func (ss *SimpleStore) schemaInit() {
	ss.metadata.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("table.")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(v []byte) error {
				var s Schema
				msgp.Decode(bytes.NewReader(v), &s)
				ss.schema[s.TableName] = s
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (ss *SimpleStore) Init(storeName string) {
	var err error
	ss.schema = make(map[string]Schema)
	ss.metadata, err = badger.Open(badger.DefaultOptions("/tmp/metadata_" + storeName))
	if err != nil {
		log.Fatal(err)
	}
	ss.data, err = badger.Open(badger.DefaultOptions("/tmp/data_" + storeName))

	ss.schemaInit()
}

func (ss *SimpleStore) Shutdown() {
	ss.metadata.Close()
}

func (ss *SimpleStore) GetSchemas() []Schema {
	var s []Schema
	for _, v := range ss.schema {
		s = append(s, v)
	}
	return s
}
func (ss *SimpleStore) create(q query.Query) error {
	if _, exists := ss.schema[q.TableName]; exists {
		return errors.New("Table '" + q.TableName + "' already exists")
	}
	var schema Schema
	schema.Init(q.TableName, q.Fields, q.DTypes)
	var b bytes.Buffer
	msgp.Encode(&b, &schema)
	err := ss.metadata.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("table."+schema.TableName), b.Bytes())
	})
	if err != nil {
		return err
	}
	ss.schema[schema.TableName] = schema
	return err
}

func (ss *SimpleStore) insert(q query.Query) error {
	schema, ok := ss.schema[q.TableName]
	if !ok {
		return errors.New(`No table with name '{{q.TableName}}'`)
	}
	err := schema.ValidateFields(q.Fields)
	if err != nil {
		return err
	}
	entries := make(chan Entry, 5)
	var wg sync.WaitGroup
	go schema.Write(q.Fields, q.Inserts, entries, &wg)
	seq, err := ss.data.GetSequence([]byte("auto_increment."+schema.TableName), 1000)
	defer seq.Release()

	err = ss.data.Update(func(txn *badger.Txn) error {
		for c := range entries {
			if c.Err != nil {
				txn.Discard()
				return c.Err
			}
			seqNum, err := seq.Next()
			if err != nil {
				txn.Discard()
				return err
			}
			err = txn.Set([]byte(fmt.Sprintf("%s.%d", schema.TableName, seqNum)), c.Value)
			if err != nil {
				txn.Discard()
				return err
			}
			wg.Done()
		}
		return nil
	})
	return err
}

func (ss *SimpleStore) qselect(q query.Query) (*QueryResult, error) {
	schema, ok := ss.schema[q.TableName]
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
		ss.create(q)
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
