# Motivations

This project aims to provide a better environment for building hardware projects with complexe interactions between the low level (micro-controllers/embedded software/...) and the higher level (eg. UI/UX for public access).
It also aims to resolves many of the common difficulties and caveats of modern embedded development.

From previous observations, here is an unexhaustive list of the key concepts that should be obtained by using the rotonde project.

- **Language-agnostic**. In a typical case, where the core of the architecture is usually a raspberryPi-like computer, there is no "way-to-go" like on iOS or android (which both have a forced unique language), this freedom of choice implies that useful pieces of codes or language bindings that you gather on the WEB can be of any language. The framework should not be tighted to a specific language, and if multiple languages are used, the usability of the framework should still feel native in all languages.
- **Connected**. Building electronics in 2015 usually implies a connection to the WEB, which means that the framework give you the ability to seemlessly and securely add cloud code execution in the loop.
- **Modular**. This framework has its origins in agencies, fastness of execution is a key feature of the framework. The framework should encourage a modular architecture, which allows highly efficient reuse of code from one project to another.
- **Hardware abstraction**. One of the issue that you face when developping on embedded linux environment is the lack of unicity in the low level hardware access. For example, GPIO are not accessed the same way on a raspberryPI and an edison. The framework should greatly simplify the transfer of code from one architecture to another, without having to refactor existing code.
- **UI friendly**. The whole system should be easily connected to a screen or any HID device, even a smartphone through wireless means, or both. any combination of WEB/touch screen/external device should be greatly simplified and interchangeable.
- **Remotely debuggable**. Building custom devices implies remote support and debugging. The framework should provide an environment that is easily debuggable, even from a remote access. All modules should be individually observable, and interaction from a remote point is the key to cheap and responsive support.
- **Cluster creation**. Inter connectivity of devices is a known complexe situation. This should be easily challenged by providing a clean way of creating meshes of devices. TODO check MQTT for amazon for IoT plateform

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
./rotonde -port 4242
```

## JSON protocol

In most case, rotonde is used through its websocket (Rest interface is foreseen), by sending and receiving JSON objects.
There a five types of json objects, "update", "req", "cmd", "sub" or "unsub",
which are detailed below.

These four json objects all share a common structure :

```
{
  type: "", // "update", "req", "cmd", "sub" or "unsub"
  payload: {
    // mixed, based on type
  }
}
```

### Update

The "update" object encapsulate an update of a Object, which can be
found in two different contexts.

- Received as a notification that a setting had been updated.
- Sent to update a setting

For example, the attitude module (which is responsible for attitude estimation, which means "what is the current angle of the drone") will periodically send the quaternion representing the current angle of the drone through the AttitudeActual object.

But "update" objects can also be used to set setting values for the desired module, for example, if you send a [AttitudeSettings](https://raw.githubusercontent.com/TauLabs/TauLabs/next/shared/uavobjectdefinition/attitudesettings.xml) update object through websocket it will configure the PID algorithm that enables your drone to stay still in the air.

```
{
  "type": "update",
  "payload": {
    // objectId and instanceId are both required
    "objectId": 1234, // displayed on start of rotonde
    "instanceId": 0, // see UAVTalk documentation for info
    "data": {
      // Object data, as described by the definitions
    }
  }
}
```

### Req

Some Objects are sent periodically, like the AttitudeActual that is sent every 100 ms, but others have different update policies, for example, the AttitudeSettings object is sent when changed, which means if you want its value you can either wait for it to change (which should not occure in normal condition), or just request it by sending a "req" object into the pipe, the response will be received as a "update" object.

```
{
  "type": "req",
  "payload": {
    "objectId": 1234, // displayed on start of rotonde, will be received from the def packet
    "instanceId": 0, // see UAVTalk documentation for info
  }
}
```

### Sub / Unsub

When you connect to rotonde nothing will be received except definitions, you have to subscribe to a given objectId in order to start receiving its updates.

```
{
  "type": "sub",
  "payload": {
    "objectId": 1234 // displayed on start of rotonde, will be received from the def packet
  }
}
```

and you can unsubscribe from this objectId with:

```
{
  "type": "unsub",
  "payload": {
    "objectId": 1234 // displayed on start of rotonde, will be received from the def packet
  }
}
```

### Def

Each Object has a set of fields and meta datas, when a Object is available (like GPS), the module providing this feature sends its definition to rotonde which then dispatches a definition to the clients.
Given that a Object reflects an available feature of the drone, definitions give clients a clear overview of the available features.
A client can send definitions to rotonde, exposing the feature that it provides.

When you connect to rotonde, it will start be sending you all the currently available definitions, new definitions can still become available at any time.

```
{
  "type": "def",
  "payload": {
    // meta datas from Object, at first will be tightly linked to definitions found in the xml files
  }
}
```

#Licence

[Apache licence 2.0 ](https://github.com/HackerLoop/rotonde/blob/master/licence.md)
