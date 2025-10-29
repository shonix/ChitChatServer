# ChitChat - Distributed Chat Service

A distributed chat system implementing Lamport logical timestamps for causal event ordering. Built with Go and gRPC.

## Prerequisites

- Go 1.25.0 or later

## Running the System

Start the server in one terminal:
```bash
go run server/server.go
```

Start clients in separate terminals (minimum 3 clients):
```bash
go run client/client.go
```

Enter a username when prompted. Type messages to chat, or type `exit` to leave.

## System Features

- Lamport logical timestamps for event ordering
- Bidirectional gRPC streaming between clients and server
- Structured logging (format: `[Component] Timestamp Event=Type Key=Value`)
- Message validation (max 128 UTF-8 chars)
- Shutdown and leave notifications