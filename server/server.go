package main

import (
	proto "ChitChat/grpc"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type ChatServer struct {
	proto.UnimplementedChitChatServer
	streams map[string]proto.ChitChat_ChatServer
}

func (s *ChatServer) Chat(stream proto.ChitChat_ChatServer) error {
	// Identify this client
	p, _ := peer.FromContext(stream.Context())
	clientID := p.Addr.String()

	s.streams[clientID] = stream
	log.Printf("Client connected: %s\n", clientID)

	// Listen for incoming messages
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Client disconnected: %s\n", clientID)
			delete(s.streams, clientID)
			return err
		}

		log.Printf("Received from %s: %s\n", clientID, msg.Message)
		s.broadcast(msg, clientID)
	}
}

func (s *ChatServer) broadcast(msg *proto.ClientMessage, sender string) {
	for id, st := range s.streams {
		if err := st.Send(msg); err != nil {
			log.Printf("Failed to send to %s: %v", id, err)
		}
	}
}

func main() {
	server := &ChatServer{
		streams: make(map[string]proto.ChitChat_ChatServer),
	}

	lis, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChitChatServer(grpcServer, server)

	log.Println("gRPC Chat server running on :5050 ...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
