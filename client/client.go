package main

import (
	proto "ChitChat/grpc"
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working")
	}

	client := proto.NewChitChatClient(conn)

	stream, err := client.Chat(context.Background())
	if err != nil {
		log.Fatalf("Not working")
	}
	// Example: send a message to server
	msg := &proto.ClientMessage{
		Message: "Hello from client!",
	}

	if err := stream.Send(msg); err != nil {
		log.Fatalf("Send error: %v", err)
	}
	log.Println("Message sent!")
}
