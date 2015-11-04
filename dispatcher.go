package main

import (
	"reflect"

	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
)

// ChanQueueLength buffered channel length
const ChanQueueLength = 10

// Connection : basic interface representing a connection to the dispatcher
type Connection struct {
	actions rotonde.Definitions // actions that this connection can receive
	events  rotonde.Definitions // events that this connection can send

	subscriptions []string

	InChan  chan interface{}
	OutChan chan interface{}
}

// NewConnection creates a new dispatcher connection
func NewConnection() *Connection {
	connection := new(Connection)

	connection.InChan = make(chan interface{}, ChanQueueLength)
	connection.OutChan = make(chan interface{}, ChanQueueLength)

	return connection
}

func (connection *Connection) Close() {
	close(connection.OutChan)
	close(connection.InChan)
}

func (connection *Connection) addSubscription(identifier string) {
	if connection.isSubscribed(identifier) {
		return
	}
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

func (connection *Connection) isSubscribed(identifier string) bool {
	for _, subscription := range connection.subscriptions {
		if subscription == identifier {
			return true
		}
	}
	return false
}

type Dispatcher struct {
	definitionsLog map[string]int // this is to keep track of the
	definitions    rotonde.Definitions
	//available definitions, it maps identifiers to the number of time a
	//definition has been declared by a module, and dispatched defs and
	//undefs packets when needed (please see comments above the
	//dispatchDefinition function).
	connections    []*Connection
	cases          []reflect.SelectCase // cases for the select case of the main loop, the first element is for the connectionChan, the others are for the outChans of the connections
	connectionChan chan *Connection     // connectionChan receives the new connections to add
}

func NewDispatcher() *Dispatcher {
	dispatcher := new(Dispatcher)
	dispatcher.connections = make([]*Connection, 0, 100)
	dispatcher.cases = make([]reflect.SelectCase, 0, 100)
	dispatcher.connectionChan = make(chan *Connection, 10) // TODO try unbuffered chan
	dispatcher.definitionsLog = make(map[string]int)
	dispatcher.definitions = make([]*rotonde.Definition, 0, 100)

	// first case is for the connectionChan
	dispatcher.cases = append(dispatcher.cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(dispatcher.connectionChan)})

	return dispatcher
}

func (dispatcher *Dispatcher) AddConnection(connection *Connection) {
	dispatcher.connectionChan <- connection
}

func (dispatcher *Dispatcher) addConnection(connection *Connection) {
	for _, def := range dispatcher.definitions {
		connection.InChan <- *def
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

func (dispatcher *Dispatcher) dispatchEvent(from int, event *rotonde.Event) {
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

func (dispatcher *Dispatcher) dispatchAction(from int, action *rotonde.Action) {
	for i, connection := range dispatcher.connections {
		if i == from {
			continue
		}

		if _, err := connection.actions.GetDefinitionForIdentifier(action.Identifier); err == nil {
			connection.InChan <- *action
		}
	}
}

// Edit: I reverted the commented code, the idea now it to thrown warnings (or errors?)
// when multiple definitions with same identifier have different fields
// TODO create field unicity check
// TODO check if no same identifier in action an event
//
// The code commented in the two next functions changes the behaviour
// of rotonde when facing multiple modules declaring the same
// definition (eg. with same identifiers). Currently, all def and undef packets are sent, the module
// has to be able to determine if the action or event it requires is of
// the right structure.
// This is actually part of a larger discussion about version management
// for modules, and identifier collision.
//
// The commented code below restricts the dispatch of definition packes
// to the first added and last removed for a given identifier.
//
// A more normal way of doing would to also take into account the
// fields, and change the rule to: only send def packets when its the
// first time a definition with this identifier AND fields is declared.
// This would involve generating a hash to quickly match the fields.
//
// This could be the right thing to do, but it needs to be well thought
// as it could force to do this distinction in other places. In sub
// packets for example.
//
// So the simplest now is to just let the modules check if the
// definition and actions/events are given with the right structure.

func (dispatcher *Dispatcher) dispatchDefinition(from int, definition *rotonde.Definition) {
	if n, ok := dispatcher.definitionsLog[definition.Identifier]; ok == true && n >= 1 {
		n++
		dispatcher.definitionsLog[definition.Identifier] = n
		return // we only send def packets when it is a new packet (when ok == false or n == 0)
	}
	dispatcher.definitionsLog[definition.Identifier] = 1
	dispatcher.definitions = rotonde.PushDefinition(dispatcher.definitions, definition)
	for i, connection := range dispatcher.connections {
		if i == from {
			continue
		}
		connection.InChan <- *definition
	}
}

func (dispatcher *Dispatcher) dispatchUnDefinition(from int, unDefinition *rotonde.UnDefinition) {
	if n, ok := dispatcher.definitionsLog[unDefinition.Identifier]; ok == false {
		log.Warning("calling dispatchUnDefinition with a definition that has never been registered")
		return
	} else {
		n--
		dispatcher.definitionsLog[unDefinition.Identifier] = n
		if n >= 1 {
			return // we only send undef packets when n reaches zero
		}
	}
	dispatcher.definitions = rotonde.RemoveDefinition(dispatcher.definitions, unDefinition.Identifier)
	for i, connection := range dispatcher.connections {
		if i == from {
			continue
		}
		connection.InChan <- *unDefinition
	}
}

func (dispatcher *Dispatcher) processChannels() {
	chosen, value, ok := reflect.Select(dispatcher.cases)
	chosen-- // there is an offset of 1, because the first element of dispatcher.cases is for the connection chan
	if !ok {
		log.Warning("One of the channels is broken.", chosen)
		for _, definition := range append(dispatcher.connections[chosen].actions, dispatcher.connections[chosen].events...) {
			log.Info("Dispatching UnDefinition message")
			unDefinition := rotonde.UnDefinition(*definition)
			dispatcher.dispatchUnDefinition(chosen, &unDefinition)
		}
		dispatcher.removeConnectionAt(chosen)
	} else {
		switch data := value.Interface().(type) {
		case rotonde.Event:
			log.Info("Dispatching event")
			dispatcher.dispatchEvent(chosen, &data)
		case rotonde.Action:
			log.Info("Dispatching action")
			dispatcher.dispatchAction(chosen, &data)
		case rotonde.Subscription:
			log.Info("Executing subscribe")
			connection := dispatcher.connections[chosen]
			connection.addSubscription(data.Identifier)
		case rotonde.Unsubscription:
			log.Info("Executing unsubscribe")
			connection := dispatcher.connections[chosen]
			connection.removeSubscription(data.Identifier)
		case rotonde.Definition:
			log.Info("Dispatching Definition message")
			connection := dispatcher.connections[chosen]
			if data.Type == "action" {
				connection.actions = rotonde.PushDefinition(connection.actions, &data)
			} else if data.Type == "event" {
				connection.events = rotonde.PushDefinition(connection.events, &data)
			}
			dispatcher.dispatchDefinition(chosen, &data)
		case rotonde.UnDefinition:
			log.Info("Dispatching UnDefinition message")
			connection := dispatcher.connections[chosen]
			var definition *rotonde.Definition
			if data.Type == "action" {
				def, err := connection.actions.GetDefinitionForIdentifier(data.Identifier)
				if err != nil {
					log.Warning(err)
					break
				}
				definition = def
				connection.actions = rotonde.RemoveDefinition(connection.actions, data.Identifier)
			} else if data.Type == "event" {
				def, err := connection.events.GetDefinitionForIdentifier(data.Identifier)
				if err != nil {
					log.Warning(err)
					break
				}
				definition = def
				connection.events = rotonde.RemoveDefinition(connection.events, data.Identifier)
			}
			unDefinition := rotonde.UnDefinition(*definition)
			dispatcher.dispatchUnDefinition(chosen, &unDefinition)
		case *Connection:
			log.Info("Add connection")
			dispatcher.addConnection(data) // data is already a pointer
		default:
			log.Warning("Oops got some unknown object in the dispatcher, ignoring.")
		}
	}
}

func (dispatcher *Dispatcher) Start() {
	for {
		dispatcher.processChannels()
	}
}
