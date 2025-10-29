package main

import (
	proto "ChitChat/grpc"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"google.golang.org/grpc"
)

type ChatServer struct {
	proto.UnimplementedChitChatServer
	mu             sync.RWMutex
	streams        map[string]proto.ChitChat_ChatServer
	lamportClock   int64
	clockMu        sync.Mutex
}

func main() {
	log.SetPrefix("[Server] ")
	log.SetFlags(log.Ldate | log.Ltime)

	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatalf("Event=ServerStartupFailed Error=%v", err)
	}

	grpcServer := grpc.NewServer()
	chatServer := &ChatServer{
		streams:      make(map[string]proto.ChitChat_ChatServer),
		lamportClock: 0,
	}

	proto.RegisterChitChatServer(grpcServer, chatServer)
	log.Printf("Event=ServerStartup Status=Success Port=5050")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Printf("Event=ServerShutdown Status=Initiated")
		grpcServer.GracefulStop()
		log.Printf("Event=ServerShutdown Status=Completed")
		os.Exit(0)
	}()

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Event=ServerStartupFailed Error=%v", err)
	}
}

func (s *ChatServer) incrementClock() int64 {
	s.clockMu.Lock()
	defer s.clockMu.Unlock()
	s.lamportClock++
	return s.lamportClock
}

func (s *ChatServer) updateClock(receivedTime int64) int64 {
	s.clockMu.Lock()
	defer s.clockMu.Unlock()
	if receivedTime > s.lamportClock {
		s.lamportClock = receivedTime
	}
	s.lamportClock++
	return s.lamportClock
}

func (s *ChatServer) Chat(stream proto.ChitChat_ChatServer) error {
	var username string

	// Receive connection message and set username
	connMsg, err := stream.Recv()
	if err != nil {
		log.Printf("Event=ClientConnectionFailed Error=%v", err)
		return err
	}

	// Update Lamport clock on receiving join message
	currentTime := s.updateClock(connMsg.LamportTimestamp)

	username = connMsg.From
	if username == "" {
		username = "guest"
	}

	// Register client
	s.mu.Lock()
	s.streams[username] = stream
	s.mu.Unlock()

	log.Printf("Event=ClientConnected ClientID=%s LamportTime=%d", username, currentTime)

	// Create join message with updated Lamport timestamp
	joinMsg := &proto.ClientMessage{
		LamportTimestamp: currentTime,
		From:             "server",
		Message:          fmt.Sprintf("Participant %s joined Chit Chat at Lamport time %d", username, currentTime),
	}

	// Broadcast join message to ALL participants (including the new one)
	s.broadcastToAll(joinMsg)
	log.Printf("Event=BroadcastJoin ClientID=%s LamportTime=%d", username, currentTime)

	// Handle client disconnection
	defer func() {
		s.mu.Lock()
		delete(s.streams, username)
		s.mu.Unlock()

		leaveTime := s.incrementClock()
		log.Printf("Event=ClientDisconnected ClientID=%s LamportTime=%d", username, leaveTime)

		leaveMsg := &proto.ClientMessage{
			LamportTimestamp: leaveTime,
			From:             "server",
			Message:          fmt.Sprintf("Participant %s left Chit Chat at Lamport time %d", username, leaveTime),
		}
		s.broadcast(leaveMsg, username)
		log.Printf("Event=BroadcastLeave ClientID=%s LamportTime=%d", username, leaveTime)
	}()

	// Receive messages from client
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		if strings.TrimSpace(msg.Message) == "" {
			continue
		}

		// Validate message length (128 characters max)
		if len(msg.Message) > 128 {
			log.Printf("Event=MessageRejected ClientID=%s Reason=ExceedsMaxLength Length=%d", username, len(msg.Message))
			continue
		}

		// Update Lamport clock with received timestamp
		msgTime := s.updateClock(msg.LamportTimestamp)
		msg.LamportTimestamp = msgTime

		log.Printf("Event=MessageReceived ClientID=%s LamportTime=%d Message=%s", msg.From, msgTime, msg.Message)
		s.broadcast(msg, username)
		log.Printf("Event=MessageBroadcast ClientID=%s LamportTime=%d Recipients=%d", msg.From, msgTime, len(s.streams)-1)
	}
}

func (s *ChatServer) broadcast(msg *proto.ClientMessage, sender string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, clientStream := range s.streams {
		if name == sender {
			continue
		}
		if err := clientStream.Send(msg); err != nil {
			log.Printf("Event=BroadcastError ClientID=%s Error=%v", name, err)
		}
	}
}

func (s *ChatServer) broadcastToAll(msg *proto.ClientMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, clientStream := range s.streams {
		if err := clientStream.Send(msg); err != nil {
			log.Printf("Event=BroadcastError ClientID=%s Error=%v", name, err)
		}
	}
}
