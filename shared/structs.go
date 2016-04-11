package rotonde

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
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

func PushDefinition(definitions Definitions, def *Definition) Definitions {
	if _, err := definitions.GetDefinitionForIdentifier(def.Identifier); err == nil {
		return definitions
	}
	return append(definitions, def)
}

func RemoveDefinition(definitions Definitions, identifier string) Definitions {
	for i, definition := range definitions {
		if definition.Identifier == identifier {
			if i < len(definitions)-1 {
				copy(definitions[i:], definitions[i+1:])
			}
			definitions = definitions[0 : len(definitions)-1]
			return definitions
		}
	}
	return definitions
}

// Fields sortable slice of fields
type FieldDefinitions []*FieldDefinition

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
	IsArray    bool   `json:"isarray"`

	Fields FieldDefinitions `json:"fields"`
}

func (d *Definition) PushField(n, t, u string) {
	field := FieldDefinition{n, t, u}
	d.Fields = append(d.Fields, &field)
}

type UnDefinition Definition

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

func ToJSON(object interface{}) ([]byte, error) {
	var packet Packet
	switch data := object.(type) {
	case Event:
		packet = Packet{Type: "event", Payload: data}
	case Action:
		packet = Packet{Type: "action", Payload: data}
	case Subscription:
		packet = Packet{Type: "sub", Payload: data}
	case Unsubscription:
		packet = Packet{Type: "unsub", Payload: data}
	case Definition:
		packet = Packet{Type: "def", Payload: data}
	case UnDefinition:
		packet = Packet{Type: "undef", Payload: data}
	default:
		log.Fatal("Oops unknown packet: ", object)
	}

	jsonPacket, err := json.Marshal(packet)
	if err != nil {
		return nil, err
	}

	return jsonPacket, nil
}

func FromJSON(reader io.Reader) (interface{}, error) {
	packet := Packet{}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&packet); err != nil {
		return nil, err
	}

	switch packet.Type {
	case "event":
		event := Event{}
		mapstructure.Decode(packet.Payload, &event)
		return event, nil
	case "action":
		action := Action{}
		mapstructure.Decode(packet.Payload, &action)
		return action, nil
	case "sub":
		subscription := Subscription{}
		mapstructure.Decode(packet.Payload, &subscription)
		return subscription, nil
	case "unsub":
		unsubscription := Unsubscription{}
		mapstructure.Decode(packet.Payload, &unsubscription)
		return unsubscription, nil
	case "def":
		definition := Definition{}
		mapstructure.Decode(packet.Payload, &definition)
		return definition, nil
	case "undef":
		unDefinition := UnDefinition{}
		mapstructure.Decode(packet.Payload, &unDefinition)
		return unDefinition, nil
	}
	return nil, fmt.Errorf("%s not found", packet.Type)
}
