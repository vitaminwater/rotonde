package common

import (
	"errors"
	"fmt"
)

// Definitions is a slice of Definition, adds findBy
type Definitions []*Definition

// GetDefinitionForObjectID _
func (definitions Definitions) GetDefinitionForObjectID(objectID uint32) (*Definition, error) {
	for _, definition := range definitions {
		if definition.ObjectID == objectID {
			return definition, nil
		}
	}
	return nil, errors.New(fmt.Sprint(objectID, " Not found"))
}

// GetDefinitionForName _
func (definitions Definitions) GetDefinitionForName(name string) (*Definition, error) {
	for _, definition := range definitions {
		if definition.Name == name {
			return definition, nil
		}
	}
	return nil, errors.New(fmt.Sprint(name, " Not found"))
}

// FieldsSlice sortable slice of fields
type FieldsSlice []*FieldDefinition

// FieldForName returns a fieldDefinition for a given name
func (fields FieldsSlice) FieldForName(name string) (*FieldDefinition, error) {
	for _, field := range fields {
		if field.Name == name {
			return field, nil
		}
	}
	return nil, fmt.Errorf("Not found field name: %s", name)
}

// FieldDefinition _
type FieldDefinition struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Units string `json:"units"`
}

// Definition _
type Definition struct {
	ObjectID uint32 `json:"id" mapstructure:"id"`
	Name           string `json:"name"`

	Fields FieldsSlice `json:"fields"`
}
