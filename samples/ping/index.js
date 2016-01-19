'use strict';

let _ = require('lodash');
let WebSocket = require('ws');

let ws = new WebSocket('ws://localhost:4224/');

ws.on('open', () => {
  console.log('connected');
  ws.send(JSON.stringify({
    type: 'sub',
    payload: {
      identifier: 'PONG',
    }
  }));
  sendPing(0);
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
      if (packet.payload.identifier == 'PONG') {
        sendPing(packet.payload.data.n);
      }
    break;
    case 'action':
      console.log('received action', packet.payload);
    break;
  }
});

function sendPing(n) {
  ws.send(JSON.stringify({
    type: 'action',
    payload: {
      identifier: 'PING',
      data: {
        n: n + 1,
      }
    },
  }));
}
