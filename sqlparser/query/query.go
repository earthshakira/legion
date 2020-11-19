package query

import (
	"bytes"
	"encoding/json"
)

// Query represents a parsed query
type Query struct {
	Type       Type
	TableName  string
	Conditions []Condition
	Updates    map[string]string
	Inserts    [][]string
	Fields     []string // Used for SELECT (i.e. SELECTed field names) and INSERT (INSERTEDed field names)
	DTypes     []DType
}

// Type is the type of SQL query, e.g. SELECT/UPDATE
type Type int

const (
	// UnknownType is the zero value for a Type
	UnknownType Type = iota
	// Select represents a SELECT query
	Select
	// Update represents an UPDATE query
	Update
	// Insert represents an INSERT query
	Insert
	// Delete represents a DELETE query
	Delete
	// Create represents a CREATE TABLE query
	Create
)

// Type is the type of SQL query, e.g. SELECT/UPDATE
type DType int

const (
	// UnknownDtype for the database
	UnknownDtype DType = iota
	// Bool for the database
	Bool
	// Int for the database
	Int
	// Double for the database
	Double
	// String for the database
	String
	// DateTime for the database
	DateTime
	// Geo2d for the database
	Geo2d
)

var dtypeToString = map[DType]string{
	UnknownDtype: "UnknownDtype",
	Bool:         "Bool",
	Int:          "Integer",
	Double:       "Double",
	String:       "String",
	DateTime:     "DateTime",
	Geo2d:        "Geo2d",
}

var stringToDtype = map[string]DType{
	"UnknownDtype": UnknownDtype,
	"Bool":         Bool,
	"Integer":      Int,
	"Double":       Double,
	"String":       String,
	"DateTime":     DateTime,
	"Geo2d":        Geo2d,
}

func (s DType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(dtypeToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *DType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*s = stringToDtype[j]
	return nil
}

// TypeString is a string slice with the names of all types in order
var TypeString = []string{
	"UnknownType",
	"Select",
	"Update",
	"Insert",
	"Delete",
}

// Operator is between operands in a condition
type Operator int

const (
	// UnknownOperator is the zero value for an Operator
	UnknownOperator Operator = iota
	// Eq -> "="
	Eq
	// Ne -> "!="
	Ne
	// Gt -> ">"
	Gt
	// Lt -> "<"
	Lt
	// Gte -> ">="
	Gte
	// Lte -> "<="
	Lte
)

// OperatorString is a string slice with the names of all operators in order
var OperatorString = []string{
	"UnknownOperator",
	"Eq",
	"Ne",
	"Gt",
	"Lt",
	"Gte",
	"Lte",
}

// Condition is a single boolean condition in a WHERE clause
type Condition struct {
	// Operand1 is the left hand side operand
	Operand1 string
	// Operand1IsField determines if Operand1 is a literal or a field name
	Operand1IsField bool
	// Operator is e.g. "=", ">"
	Operator Operator
	// Operand1 is the right hand side operand
	Operand2 string
	// Operand2IsField determines if Operand2 is a literal or a field name
	Operand2IsField bool
}
