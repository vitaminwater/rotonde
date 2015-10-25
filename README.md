# Motivations

This project aims to resolves many of the common difficulties and caveats of modern embedded development.

From previous observations, here is an unexhaustive list of the key concepts that should be obtained by using the `rotonde` project.

- **Language-agnostic**. In a typical case, where the core of the architecture is usually a raspberryPi-like computer, there is no "way-to-go" like on iOS or android (which both have a forced unique language), this freedom of choice implies that useful pieces of codes or language bindings that you gather on the WEB can be of any language. The framework should not be tighted to a specific language, and if multiple languages are used, the usability of the framework should still feel native in all languages.
- **Connected**. Building electronics in 2015 usually implies a connection to the WEB, which means that the framework give you the ability to seemlessly and securely add cloud code execution in the loop.
- **Modular**. This framework has its origins in agencies, fastness of execution is a key feature of the framework. The framework should encourage a modular architecture, which allows highly efficient reuse of code from one project to another.
- **Hardware abstraction**. One of the issue that you face when developping on embedded linux environment is the lack of unicity in the low level hardware access. For example, GPIO are not accessed the same way on a raspberryPI and an edison. The framework should greatly simplify the transfer of code from one architecture to another, without having to refactor existing code.
- **UI friendly**. The whole system should be easily connected to a screen or any HID device, even a smartphone through wireless means, or both. any combination of WEB/touch screen/external device should be greatly simplified and interchangeable.
- **Remotely debuggable**. Building custom devices implies remote support and debugging. The framework should provide an environment that is easily debuggable, even from a remote access. All modules should be individually observable, and interaction from a remote point is the key to cheap and responsive support.
- **Cluster creation**. Inter connectivity of devices is a known complexe situation. This should be easily challenged by providing a clean way of creating meshes of devices. TODO check MQTT for amazon for IoT plateform


# Rotonde

When you code with `rotonde` you have to think in term of modules.
Each module is connected to `rotonde`, and communicate with each other
though `rotonde`.

## Events and actions

In `rotonde` each module exposes its API through `events` and `actions`.
`Events` and `actions` have similarities in their behavior, but they
represent two different concepts.

`Events` are a way for modules to send a message to the rest of the
system. These `event` can be sent spontaneously, or as a reply to an
`action`.

`Actions` are a way for modules to expose their features to the rest of
the system. Other module can then use this action to tell the module
what to do.

Declaring an `event` or `action` to rotonde is achieved by sending a `def`
packet, please see the [def](https://github.com/HackerLoop/rotonde#def)
section below.

When a module exposes an `action` or `event` to rotonde, it has to specify an
identifier and a list of fields. The fields are mostly here as a small
documentation for the developper and as a way for the other modules to
make sure that the action has the desired fields.

The first thing that a module receives when connecting to `rotonde` is the list
of all `events` and `actions` that have been defined on rotonde by other
modules.
This gives modules the opportunity to check that the system they are
connected to has the required features available.

## Events and actions routing

Internally rotonde works like some kind of a router or dispatcher.
The routing rules are slightly different for actions and events.

When rotonde receives an action from a module, it looks at all modules
that exposed this actions, and dispatches the action to them.

When rotonde receives an `event` from a module, it only dispatches it to
the modules that subscribed to this `event`.

# Setup 

## Requirements 

- some unix os (tested with success on Linux and OSX so far)
- [Golang](https://golang.org/) (1.5.1, please tell us if you got it
  working on previous versions, we didn't test them yet)
- [Godep](http://godoc.org/github.com/tools/godep)

## Compilation

Assuming Golang had been installed, if it's not already done a workspace
can be set with  

```bash
export GOPATH=$HOME/go
mkdir $GOPATH
go get github.com/HackerLoop/rotonde && go get github.com/tools/godep
cd $GOPATH/src/github.com/HackerLoop/rotonde
godep restore
go build
```

`go build` will compile an executable called `rotonde` in the project
folder (`$GOPATH/src/HackerLoop/rotonde`).

If something goes wrong during compilation, please [open up an
issue](https://github.com/HackerLoop/rotonde/issues) with your
os/distribution infos and compilation output. 

## Running

```bash
./rotonde
```

Rotonde will start serving on port 4224 by default, add option `-port`
to specify another one.

# JSON protocol

In most case, rotonde is used through its websocket (other interfaces are foreseen), by sending and receiving JSON objects.

All objects received or sent from/to rotonde have this structure:

```
{
  type: "",
  payload: {
    // mixed, based on type
  }
}
```

There a five possible values for the `type` field, `event`, `action`, `def`, `sub` or `unsub`,
the content of the `payload` field varies based on the type.

The different possible structures of the payload are described below.

### Def

When a module connects to rotonde, it has to detail its API to rotonde.
It does so by sending `definition` objects, rotonde routes all received
definitions to all connected modules. And when you connect to rotonde,
you receive all the available definitions.


Knowing that everything is either an action or an event in rotonde,
there are two types of definitions, either `action` or `event`.


Each definitions contain a set of fields describing its structure. This
is mainly to ensure that the action or event is present with a
predictable structure.


In a typical scenario, a module that connects to rotonde starts by
waiting for all the events and actions it requires to work properly.
Definitions are a sort of description of what is available on the
system.


This way of doing things gives you the ability to create a modular
system where some module would only start when a given capability is
present.


```
{
  "identifier": "",
  "type": "",
  "fields":[
    {
      "name": "",
      "type": "", // optional
      "unit": "", // optional
    },
    {
      ... other fields ...
    }
  ]
}
```


### Action


In rotonde, everything is either an action or an event, they are the
only way for the modules to exchange data with the external world.


Actions are the APIs of the modules, each action has an identifier; when
a module declares an action through the mean of a `definition` object,
it will receive all actions matching this identifier sent to rotonde.
If multiple modules declare the same action, they will all receive it.


Actions are typically sent by user interface modules, for example, when
a user presses a button, the controller of the button will send an action,
that will be handled by one or multiple modules.
For example, if the button is meant to switch a light on, the action
identifier would be `TURN_LIGHT_ON`, this could totally trigger the
light control module, but if we want to play a music when this happens,
just do another module that also exposes the `TURN_LIGHT_ON` action, and
starts the music when it receives the action.


An action generally comes with some data, this data can be of any form,
its structure should be described by the definitions.


An action payload contains the following fields:

```
{
  "identifier":"",
  "data": {
    ... data fields of the action ...
  }
}
```

Actions can be seen as the input of modules.


### Event


Modules will often have things to say, whether they want to tell what
there sensor is sensing, or whether they want to report a status.


This is what events are made for. Modules have the ability to send
events through rotonde, these events will be routed to the concerned
modules.


An event generally comes with some data, this data can be of any form,
its structure should be described by the definitions.


An event payload contains the following fields:

```
{
  "identifier":"",
  "data": {
    ... data fields of the event ...
  }
}
```


Events can be seen as the output of modules.


### Sub / Unsub


When you connect to rotonde nothing will be received except definitions, you have to subscribe to a given event in order to start receiving it.

```
{
  "type": "sub",
  "payload": {
    "identifier": ""
  }
}
```

and you can unsubscribe from this identifier with:

```
{
  "type": "unsub",
  "payload": {
    "identifier": ""
  }
}
```

# Abstractions

Rotonde can be used as-is but having an abstraction above the
websocket, makes it much more efficient.

This is a list (in progress) of the available abstractions:

- [rotonde-client.js](https://github.com/HackerLoop/rotonde-client.js)

# Libraries to port to rotonde

The number of available modules is crucial for rotonde, the good news is
that it is actually quite easy to take a library and make its API
available to rotonde modules.


Here is a list of libraries that are ported, or will be ported:

- Serial:
  - [rotonde serial module](https://github.com/HackerLoop/serial-port-json-server) exposes the features of [serial-port-json-server](https://github.com/johnlauer/serial-port-json-server)(SPJS)
    - serial devices listing
    - advanced serial communication
    - flashing of an arduino from an url :) this one is crazy haha.
    - please see his README first
- Gobot:
  - This is a work in progress and a huge gain for rotonde, gobot
    already has the ability to make programs that expose features
    through a REST API, and because it is written in Go, it is really
    easy to just swap REST to websocket.

    The plan here is to generate tons of modules for all of gobot
    drivers, in all supported plateforms.

    A skeleton project is on its way to ease the work.
- TODO: expand list

# Contributing

Yes please.

![](https://d2v4zi8pl64nxt.cloudfront.net/1362775331_b8c6b6e89781c85fee638dffe341ff64.jpg)

# Licence

[Apache licence 2.0 ](https://github.com/HackerLoop/rotonde/blob/master/licence.md)
