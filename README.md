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

# Flux for embedded software

`rotonde` is a Flux architecture for embedded software developers.

If you are an embedded developer you might have missed the revolution
happening in web and application development. This field of computer
programming has been mostly dominated by the MVC (and its cousins) pattern.
I bet you know MVC, which was the perfect fit for the web development of
the past, where applications where totally constructed around the concept of
request to html response.

What Flux did was to respond to the fact that modern web applications,
and applications in general don't really fit with the request-response
pattern. Flux proposes a architecture that relies on the fact that data
has a unique flowing direction, when most modern technics involved MVC
like patterns with data bindings.

In Flux, everything is well separated, and the whole picture is only
made of 4 elements, in a typical web or mobile applications, these 4
elements are UI, actions, events and stores.

A classical scenario is when a user presses a button; when this happens,
the UI creates a `CLICKED_BUTTON` action and send it the stores through a dispatcher,
the concerned stores process the action by modifying there data, and if data has changed, the
store sends an event to notify the view, which can then display the
change made to the store.

And now the loop is complete, data flows in a unique direction: `UI => action => store => event => UI`.
For more infos on flux: see the documentation by facebook: [Here](https://facebook.github.io/flux/).

When I started using this pattern, which is great for application
development, I couldn't help but think that this could totally be applied to
embedded development.

The analogy is made by the fact that you could
totally replace the UI by any kind of input that you have when you code
on an embedded plateform, whether it's a physical hardware button, an actual UI on a touch
screen, or even a sensor.
And the stores are the different components of your platform, lets say you
are on a raspberry PI, and you want to control a stepper motor, you'd
send a `SPIN_MOTOR` action with "speed" and "duration"
as payload, then the store responsible for controlling the motors receives it
and starts the motor at the given speed for the given duration, when the
duration is reached, it sends an event `MOTOR_STOPPED` to tell the rest
of the system that everything went fine.

So what `rotonde` gives you is a kind of a `dispatcher/event emitter`
for actions and events.

The current implemented interface to this dispatcher is a websocket, but
any kind of transport layer could be implemented (let me know if you
need something in particular through an issue).

## Differences with flux

If you have carefully read the Flux documentation, you might have
noticed that it is not exactly flux, that's because modules are not
really stores, there purposes goes well beyond data storage and
logic.
In embedded software, data can be real world data when using sensors,
this does not translate well in term of storage.
On a raspberryPI, the variety of inputs, outputs and there
unpredictability implies that we keep `rotonde` as flexible as possible
in term of good or even mandatory practices.


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

# The rotonde mindset

Lets say you want to control various servo motors, through the
[pca9685](http://www.adafruit.com/product/815), the first thing you do
is look for an implementation for this device for your language; of
course you find an abstraction, but for another language, so now you
are given 2 choices, either you switch all your current dev to this language,
or you port the library into your favorite language;
which is a really classical situation, especially when coding on a raspberryPI.

With `rotonde` it would be really different, instead of looking for a
library in your particular language, what you'd be looking for is a
`rotonde` module for your device, or make your own.

So going on with the `pca9685` device, we are going to design a typical
module to work with it, universally.
Let's start by searching an implementation on github, a quick googling
gets us to this repo [https://github.com/101100/pca9685](https://github.com/101100/pca9685),
the API is clean, and the fact that it is in javascript makes really
easy to adapt to `rotonde`.

The idea of our module is to expose this library's API methods to other
modules connected to `rotonde`, which is a usual strategy when porting
libraries to `rotonde`.
For our example we are going to expose the
`setPulseRange` API call, its prototype is simple:

```
setPulseRange(channel, onStep, offStep)
```

If you recall the section above, you know that this is just an action,
which we will name `SET_PULSE_RANGE`, this action has a payload made of
the three parameters, `channel`, `onStep`, and `offStep`.

Upon connection to `rotonde`, the first that our module is going to do is
to tell the other modules about the availability of the `SET_PULSE_RANGE`
action, it does so by sending a definition. A definition is a packet
that describes an action or an event available on `rotonde`, it has
a name, and a list of payload fields.

When a definition is sent to `rotonde`, `rotonde` dispatches it to all other
modules, so they know what actions are available on the system. New
modules connecting to `rotonde` start by receiving all available actions and events.

in our case, we would end up with a definition looking like:

```
{
  "identifier": "SET_PULSE_RANGE",
  "fields": [
    {
      "name": "channel",
      "type": "number",
      "unit": "none",
    },
    {
      "name": "onStep",
      "type": "number",
      "unit": "step",
    },
    {
      "name": "offStep",
      "type": "number",
      "unit": "step",
    },
  ]
}
```

when you receive this packet when connecting to `rotonde`, you know that
the `pca9685` is available on the system, and you can now send actions to
control it.
Now that we know it is here, we want to put the servo on channel 0 to its center position,
it is as easy as sending the following json object to rotonde:

```
{
  "identifier": "SET_PULSE_RANGE",
  "payload": {
    "channel": 0,
    "onStep": 0,
    "offStep": 127,
  }
}
```

When the `pca9685` module receives this action, it calls `setPulseRange(0, 0, 127)`,
and makes the servo move to its center position.

Now the thing that needs to be well understood, is that now that this
action exists, any module connected to `rotonde`, whatever the language
its made with, can access this feature, the module does not even need to
be present on the raspberryPI actually, you could also code from your
machine, with your dev environment and your toolchain, all you have to do
is get your code to connect to the PI's IP instead of localhost.

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

## JSON protocol

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
