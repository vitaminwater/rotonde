package main

import (
	"errors"
	"fmt"
	"reflect"

	log "github.com/Sirupsen/logrus"
)

// ChanQueueLength buffered channel length
const ChanQueueLength = 10

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
	Identifier string `json:"identifier" mapstructure:"id"`
	Type       string `json:"type" mapstructure:"type"` // action or event

	Fields FieldsSlice `json:"fields"`
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

// Connection : basic interface representing a connection to the dispatcher
type Connection struct {
	actions Definitions // actions that this connection can receive
	events  Definitions // events that this connection can send

	subscriptions []string

	InChan  chan interface{}
	OutChan chan interface{}
}

// NewConnection creates a new dispatcher connection
func NewConnection() *Connection {
	connection := new(Connection)

	connection.actions = make([]*Definition, 10)
	connection.events = make([]*Definition, 10)

	connection.subscriptions = make([]string, 10)

	connection.InChan = make(chan interface{}, ChanQueueLength)
	connection.OutChan = make(chan interface{}, ChanQueueLength)

	return connection
}

// Close closes the connection, possible issues...
func (connection *Connection) Close() {
	close(connection.OutChan)
}

func (connection *Connection) addSubscription(identifier string) {
	connection.subscriptions = append(connection.subscriptions, identifier)
}

func (connection *Connection) removeSubscription(identifier string) {
	for i, subscription := range connection.subscriptions {
		if subscription == identifier {
			if i < len(connection.subscriptions)-1 {
				copy(connection.subscriptions[i:], connection.subscriptions[i+1:])
			}
			connection.subscriptions = connection.subscriptions[0 : len(connection.subscriptions)-1]
			return
		}
	}
}

// Dispatcher main dispatcher class
type Dispatcher struct {
	connections    []*Connection
	cases          []reflect.SelectCase // cases for the select case of the main loop, the first element il for the connectionChan, the others are for the outChans of the connections
	connectionChan chan *Connection     // connectionChan receives the new connections to add
}

// NewDispatcher creates a dispatcher
func NewDispatcher() *Dispatcher {
	dispatcher := new(Dispatcher)
	dispatcher.connections = make([]*Connection, 0, 100)
	dispatcher.cases = make([]reflect.SelectCase, 0, 100)
	dispatcher.connectionChan = make(chan *Connection, 10) // TODO try unbuffered chan

	// first case is for the connectionChan
	dispatcher.cases = append(dispatcher.cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(dispatcher.connectionChan)})

	return dispatcher
}

// AddConnection adds a connection to the dispatcher
func (dispatcher *Dispatcher) AddConnection(connection *Connection) {
	dispatcher.connectionChan <- connection
}

func (dispatcher *Dispatcher) addConnection(connection *Connection) {
	for _, c := range dispatcher.connections {
		for _, d := range c.actions {
			connection.InChan <- *d
		}
		for _, d := range c.events {
			connection.InChan <- *d
		}
	}

	dispatcher.connections = append(dispatcher.connections, connection)
	dispatcher.cases = append(dispatcher.cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(connection.OutChan)})
}

func (dispatcher *Dispatcher) removeConnectionAt(index int) {
	// if it is not the last element, move all next elements
	if index < len(dispatcher.connections) {
		copy(dispatcher.connections[index:], dispatcher.connections[index+1:])
		copy(dispatcher.cases[index+1:], dispatcher.cases[index+2:])
	}
	dispatcher.connections[len(dispatcher.connections)-1] = nil
	dispatcher.connections = dispatcher.connections[:len(dispatcher.connections)-1]

	dispatcher.cases = dispatcher.cases[:len(dispatcher.cases)-1]
}

func (dispatcher *Dispatcher) dispatchEvent(from int, event *Event) {
mainLoop:
	for i, connection := range dispatcher.connections {
		if i == from {
			continue
		}

		for _, identifier := range connection.subscriptions {
			if identifier == event.Identifier {
				connection.InChan <- *event
				continue mainLoop
			}
		}
	}
}

func (dispatcher *Dispatcher) dispatchAction(from int, action *Action) {
	for i, connection := range dispatcher.connections {
		if i == from {
			continue
		}

		if _, err := connection.actions.GetDefinitionForIdentifier(action.Identifier); err == nil {
			connection.InChan <- *action
		}
	}
}

func (dispatcher *Dispatcher) dispatchDefinition(from int, definition *Definition) {
	for i, connection := range dispatcher.connections {
		if i == from {
			continue
		}
		connection.InChan <- *definition
	}
}

func (dispatcher *Dispatcher) processChannels() {
	chosen, value, ok := reflect.Select(dispatcher.cases)
	if !ok {
		log.Warning("One of the channels is broken.", chosen)
		dispatcher.removeConnectionAt(chosen - 1)
	} else {
		switch data := value.Interface().(type) {
		case Event:
			dispatcher.dispatchEvent(chosen-1, &data)
		case Action:
			dispatcher.dispatchAction(chosen-1, &data)
		case Subscription:
			log.Info("Executing subscribe")
			connection := dispatcher.connections[chosen-1]
			connection.addSubscription(data.Identifier)
		case Unsubscription:
			log.Info("Executing unsubscribe")
			connection := dispatcher.connections[chosen-1]
			connection.removeSubscription(data.Identifier)
		case Definition:
			log.Info("Dispatching Definition message")
			connection := dispatcher.connections[chosen-1]
			if data.Type == "action" {
				connection.actions = append(connection.actions, &data)
			} else if data.Type == "event" {
				connection.events = append(connection.events, &data)
			}

			dispatcher.dispatchDefinition(chosen-1, &data)
		case *Connection:
			log.Info("Add connection")
			dispatcher.addConnection(data) // data is already a pointer
		default:
			log.Warning("Oops got some unknown object in the dispatcher, ignoring.")
		}
	}
}

// Start starts the dispatcher
func (dispatcher *Dispatcher) Start() {
	for {
		dispatcher.processChannels()
	}
}
