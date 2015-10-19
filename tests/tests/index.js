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

ws.on('open', () => {
  console.log('connected');
});

ws.on('message', (data) => {
  let packet = JSON.parse(data);
  switch (packet.type) {
    case 'def':
      console.log('received', packet.payload.type, 'definition', packet.payload);
      if (packet.payload.type == 'action') {
        ws.send(JSON.stringify({
          type: 'action',
          payload: defToObject(packet.payload),
        }));
      } else {
        ws.send(JSON.stringify({
          type: 'sub',
          payload: {
            identifier: packet.payload.identifier,
          }
        }));
      }
    break;
    case 'event':
      console.log('received event', packet.payload);
    break;
    case 'action':
      console.log('received action', packet.payload);
    break;
  }
});
