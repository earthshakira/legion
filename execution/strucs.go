package execution

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type ScriptPayload struct {
	Name     string    `json:"name"`
	Lang     string    `json:"lang"`
	Text     string    `json:"text"`
	Modified time.Time `json:"modified_on"`
}

func (s *ScriptPayload) Bytes() []byte {
	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.
	// dec := gob.NewDecoder(&network) // Will read from network.
	// Encode (send) the value.
	err := enc.Encode(s)
	if err != nil {
		log.Fatal("encode error:", err)
	}
	return network.Bytes()
}

func (s *ScriptPayload) FromBytes(data []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&s)
	return err
}

type ScriptOutput struct {
	Cmd    string `json:"cmd"`
	Stderr string `json:"stderr"`
	Stdout string `json:"stdout"`
}

type WorkflowBlockType int

const (
	Unknown WorkflowBlockType = iota
	Script
	JSONSplitter
	DatabaseInsert
)

type WorkflowBlock struct {
	Id     int               `json:"id"`
	Parent int               `json:"parent"`
	Type   WorkflowBlockType `json:"type"`
	Value  string            `json:"value"`
}

type WorkflowPayload struct {
	// Output of the flowy plugin
	Name     string                 `json:"name"`
	Output   map[string]interface{} `json:"output"`
	Graph    []WorkflowBlock        `json:"blocks"`
	Modified time.Time              `json:"modified_on"`
}

func (w *WorkflowPayload) Bytes() []byte {
	b, err := json.Marshal(w)
	if err != nil {
		fmt.Println(err)
	}
	return b
}

func (w *WorkflowPayload) FromBytes(data []byte) error {
	err := json.Unmarshal(data, w)
	return err
}
