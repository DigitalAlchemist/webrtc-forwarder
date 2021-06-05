# WebRTC Forwarder

Proof of concept code for tunneling arbitrary UDP traffic over WebRTC

## Example
To run, open 4 terminals.

Terminal 1:
```
go build .
# This side is the "bind" side, so it binds 127.0.0.1:10001.
./webrtc-forwarder --socket-addr 127.0.0.1:10001 --negotiator-skip-receive
# Take the "offer" from this and paste in to Terminal 2 after starting
```

Terminal 2:
```
# This is the "dial" side of the connection, which will dial 127.0.0.1:10002
./webrtc-forwarder --socket-addr 127.0.0.1:10002 --dial
# After pasting offer, copy the "answer" into terminal 1
```

Terminal 3:
```
nc -u 127.0.0.1 10001
```

Terminal 4:
```
nc -l -p 10002
```

Enter text into terminal 3 an hit "enter", which should be seen in terminal 4. Terminal 4 is unable to communicate with terminal 3 first, since it doesn't know what source port to use.

## Limitations
- Currently doesn't reconnect on disconnects.
- Only supports one connection. A future version will multiplex multiple connections.
