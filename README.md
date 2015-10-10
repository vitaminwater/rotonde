# Setup 

## Requirements 

- some unix os (tested with success on Linux and OSX so far)
- [Golang](https://golang.org/) (1.4.2, please tell us if you got it
  working on previous versions, we didn't test them yet)
- [Godep](http://godoc.org/github.com/tools/godep)

## Compilation

Assuming Golang had been installed, if it's not already done a workspace
can be set with  

```bash
export GOPATH=$HOME/go
mkdir $GOPATH
go get github.com/HackerLoop/postman && go get github.com/tools/godep
cd $GOPATH/src/github.com/HackerLoop/postman
godep restore
go build
```

`go build` will compile an executable called `postman` in the project
folder (`$GOPATH/src/HackerLoop/postman`).

If something goes wrong during compilation, please [open up an
issue](https://github.com/HackerLoop/postman/issues) with your
os/distribution infos and compilation output. 

## Running

```bash
./postman -port 4242
```

## JSON protocol

In most case, postman is used through its websocket (Rest interface is foreseen), by sending and receiving JSON objects.
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
    "objectId": 1234, // displayed on start of postman
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
    "objectId": 1234, // displayed on start of postman, will be received from the def packet
    "instanceId": 0, // see UAVTalk documentation for info
  }
}
```

### Sub / Unsub

When you connect to postman nothing will be received except definitions, you have to subscribe to a given objectId in order to start receiving its updates.

```
{
  "type": "sub",
  "payload": {
    "objectId": 1234 // displayed on start of postman, will be received from the def packet
  }
}
```

and you can unsubscribe from this objectId with:

```
{
  "type": "unsub",
  "payload": {
    "objectId": 1234 // displayed on start of postman, will be received from the def packet
  }
}
```

### Def

Each Object has a set of fields and meta datas, when a Object is available (like GPS), the module providing this feature sends its definition to postman which then dispatches a definition to the clients.
Given that a Object reflects an available feature of the drone, definitions give clients a clear overview of the available features.
A client can send definitions to postman, exposing the feature that it provides.

When you connect to postman, it will start be sending you all the currently available definitions, new definitions can still become available at any time.

```
{
  "type": "def",
  "payload": {
    // meta datas from Object, at first will be tightly linked to definitions found in the xml files
  }
}
```

#Licence

[Apache licence 2.0 ](https://github.com/HackerLoop/postman/blob/master/licence.md)
