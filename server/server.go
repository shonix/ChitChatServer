package main

import (
	proto "ChitChat/grpc"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"
)

type ChatServer struct {
	proto.UnimplementedChitChatServer
	mu      sync.RWMutex
	streams map[string]proto.ChitChat_ChatServer
}

func main() {
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatal("couldn't create listener: %v", err)
	}

	grpcServer := grpc.NewServer()
	chatServer := &ChatServer{
		streams: make(map[string]proto.ChitChat_ChatServer),
	}

	proto.RegisterChitChatServer(grpcServer, chatServer)
	log.Println("Listening on port: 5050")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal("couldn't start server: %v", err)
	}
}

func (s *ChatServer) Chat(stream proto.ChitChat_ChatServer) error {
	//Client id created based on stream/connection
	var username string

	//connection message, set username
	connMsg, err := stream.Recv()
	if err != nil {
		return err
	}
	username = connMsg.From
	if username == "" {
		username = "guest"
	}

	//Register client
	s.mu.Lock()
	s.streams[username] = stream
	s.mu.Unlock()
	log.Printf("User joined: %s", username)

	//broadcast join message
	s.broadcast(connMsg, username)

	//client disconnects
	defer func() {
		s.mu.Lock()
		delete(s.streams, username)
		s.mu.Unlock()
		log.Println("Client disconnected", username)

		leaveMsg := &proto.ClientMessage{
			Timestamp: connMsg.Timestamp,
			From:      "server",
			Message:   fmt.Sprintf("%s has left the chat", username),
		}
		s.broadcast(leaveMsg, username)
	}()

	//Recieve messages from client
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		if strings.TrimSpace(msg.Message) == "" {
			continue
		}

		fmt.Printf("[%v] %s: %s\n", msg.Timestamp.AsTime().Format("15:04:05"), msg.From, msg.Message)
		s.broadcast(msg, username)
	}
}

func (s *ChatServer) broadcast(msg *proto.ClientMessage, sender string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, clientStream := range s.streams {
		if name == sender {
			continue
		}
		_ = clientStream.Send(msg)
	}
}
