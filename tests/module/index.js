'use strict';

let _ = require('lodash');
let WebSocket = require('ws');

let ws = new WebSocket('ws://localhost:4224/');

let defToObject = (def) => _.reduce(def.fields, (v, field) => {
  switch (field.type) {
    case 'string':
      v.data[field.name] = 'string test';
    break;
    case 'number':
      v.data[field.name] = Math.random() * 4242424242;
    break;
    case 'boolean':
      v.data[field.name] = Math.random() > 0.5;
    break;
  }
  return v;
}, {identifier: def.identifier, data: {}});

let actionDef = {
  type: 'def',
  payload: {
    identifier: 'testaction',
    type: 'action',
    fields: [
      {
        name: 'field1',
        type: 'number',
        units: 'm/s',
      },
      {
        name: 'field2',
        type: 'string',
      },
      {
        name: 'field3',
        type: 'boolean',
      },
    ],
  }
};

let eventDef = {
  type: 'def',
  payload: {
    identifier: 'testevent',
    type: 'event',
    fields: [
      {
        name: 'field1',
        type: 'number',
        units: 'm/s',
      },
      {
        name: 'field2',
        type: 'string',
      },
      {
        name: 'field3',
        type: 'boolean',
      },
    ],
  }
};

ws.on('open', () => {
  console.log('connected');
  ws.send(JSON.stringify(eventDef));
  ws.send(JSON.stringify(actionDef));
});

ws.on('message', (data) => {
  let packet = JSON.parse(data);
  switch (packet.type) {
    case 'def':
      console.log('received', packet.payload.type, 'definition', packet.payload);
    break;
    case 'undef':
      console.log('received', packet.payload.type, 'unDefinition', packet.payload);
    break;
    case 'event':
      console.log('received event', packet.payload);
    break;
    case 'action':
      console.log('received action', packet.payload);
      ws.send(JSON.stringify({
        type: 'event',
        payload: defToObject(eventDef.payload),
      }));
    break;
  }
});
