package rotonde

import (
	"errors"
	"fmt"
)

// wrapper for json serialized connections
type Packet struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Definitions is a slice of Definition, adds findBy
type Definitions []*Definition

// GetDefinitionForIdentifier _
func (definitions Definitions) GetDefinitionForIdentifier(identifier string) (*Definition, error) {
	for _, definition := range definitions {
		if definition.Identifier == identifier {
			return definition, nil
		}
	}
	return nil, errors.New(fmt.Sprint(identifier, " Not found"))
}

// FieldsSlice sortable slice of fields
type FieldsSlice []*FieldDefinition

// FieldDefinition _
type FieldDefinition struct {
	Name  string `json:"name"`
	Type  string `json:"type"` // string, number or boolean
	Units string `json:"units"`
}

// Definition, used to expose an action or event
type Definition struct {
	Identifier string `json:"identifier"`
	Type       string `json:"type"` // action or event

	Fields FieldsSlice `json:"fields"`
}

func (d *Definition) PushField(n, t, u string) {
	field := FieldDefinition{n, t, u}
	d.Fields = append(d.Fields, &field)
}

// Object native representation of an event or action, just a map
type Object map[string]interface{}

type Event struct {
	Identifier string `json:"identifier"`
	Data       Object `json:"data"`
}

type Action struct {
	Identifier string `json:"identifier"`
	Data       Object `json:"data"`
}

// Subscription adds an objectID to the subscriptions of the sending connection
type Subscription struct {
	Identifier string `json:"identifier"`
}

// Unsubscription removes an objectID from the subscriptions of the sending connection
type Unsubscription struct {
	Identifier string `json:"identifier"`
}
