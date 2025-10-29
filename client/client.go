package main

import (
	proto "ChitChat/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	//Create connection to server
	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	//Create client stream
	client := proto.NewChitChatClient(conn)
	stream, err := client.Chat(context.Background())
	if err != nil {
		log.Fatalf("could not create chat stream: %v", err)
	}

	username := userLogin(stream)
	go recieveMessages(stream)
	fmt.Println()
	msgInterface(stream, username)
}

func userLogin(stream proto.ChitChat_ChatClient) string {
	var username string
	fmt.Print("Enter username: ")
	fmt.Scan(&username)

	msg := &proto.ClientMessage{
		Timestamp: timestamppb.Now(),
		From:      username,
		Message:   fmt.Sprintf("%s has joined the chat!", username),
	}
	if err := stream.Send(msg); err != nil {
		log.Fatalf("Send error: %v", err)
	}
	return username
}

func msgInterface(stream proto.ChitChat_ChatClient, username string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}

		if strings.ToLower(text) == "exit" {
			fmt.Println("Bye!")
			return
		}
		sendMsg(stream, text, username)
	}
}

func recieveMessages(stream proto.ChitChat_ChatClient) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Recieve error: %v", err)
			os.Exit(-1)
		}
		fmt.Printf("[%v] %s: %s\n", msg.Timestamp.AsTime().Format("15:04:05"), msg.From, msg.Message)
	}
}

func sendMsg(stream proto.ChitChat_ChatClient, text string, username string) {
	msg := &proto.ClientMessage{
		Timestamp: timestamppb.Now(),
		From:      username,
		Message:   text,
	}
	if err := stream.Send(msg); err != nil {
		log.Fatalf("Send error: %v", err)
	}
}
