package execution

import (
	"bytes"
	"encoding/gob"
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

func (s *ScriptPayload) FromBytes(data []byte) {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&s)
	if err != nil {
		log.Fatal("decode error:", err)
	}
}

type ScriptOutput struct {
	Cmd    string `json:"cmd"`
	Stderr string `json:"stderr"`
	Stdout string `json:"stdout"`
}
