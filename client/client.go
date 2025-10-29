package main

import (
	proto "ChitChat/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	lamportClock int64
	clockMu      sync.Mutex
	username     string
}

func (c *Client) incrementClock() int64 {
	c.clockMu.Lock()
	defer c.clockMu.Unlock()
	c.lamportClock++
	return c.lamportClock
}

func (c *Client) updateClock(receivedTime int64) int64 {
	c.clockMu.Lock()
	defer c.clockMu.Unlock()
	if receivedTime > c.lamportClock {
		c.lamportClock = receivedTime
	}
	c.lamportClock++
	return c.lamportClock
}

func main() {
	// Create connection to server
	conn, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	// Create client stream
	grpcClient := proto.NewChitChatClient(conn)
	stream, err := grpcClient.Chat(context.Background())
	if err != nil {
		log.Fatalf("could not create chat stream: %v", err)
	}

	client := &Client{
		lamportClock: 0,
	}

	client.username = userLogin(stream, client)
	log.SetPrefix(fmt.Sprintf("[Client-%s] ", client.username))
	log.SetFlags(log.Ldate | log.Ltime)

	go receiveMessages(stream, client)
	fmt.Println()
	msgInterface(stream, client)
}

func userLogin(stream proto.ChitChat_ChatClient, client *Client) string {
	var username string
	fmt.Print("Enter username: ")
	fmt.Scan(&username)

	client.username = username
	joinTime := client.incrementClock()

	msg := &proto.ClientMessage{
		LamportTimestamp: joinTime,
		From:             username,
		Message:          fmt.Sprintf("%s has joined the chat!", username),
	}
	if err := stream.Send(msg); err != nil {
		log.Fatalf("Event=SendError Error=%v", err)
	}

	log.Printf("Event=JoinSent LamportTime=%d", joinTime)
	return username
}

func msgInterface(stream proto.ChitChat_ChatClient, client *Client) {
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

		//validate message format is utf8
		if !utf8.ValidString(text) {
			fmt.Println("Error: Message is not valid UTF-8")
			log.Printf("Event=MessageRejected Reason=InvalidUtf8")
		}

		// Validate message length (128 characters max)
		if len(text) > 128 {
			fmt.Println("Error: Message exceeds 128 characters")
			log.Printf("Event=MessageRejected Reason=ExceedsMaxLength Length=%d", len(text))
			continue
		}

		sendMsg(stream, text, client)
	}
}

func receiveMessages(stream proto.ChitChat_ChatClient, client *Client) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("Event=ReceiveError Error=%v", err)
			os.Exit(-1)
		}

		// Update Lamport clock on receiving message
		currentTime := client.updateClock(msg.LamportTimestamp)

		// Display message
		fmt.Printf("[Lamport: %d] %s: %s\n", msg.LamportTimestamp, msg.From, msg.Message)

		// Log message as required
		log.Printf("Event=MessageReceived LamportTime=%d From=%s Message=%s LocalLamportTime=%d",
			msg.LamportTimestamp, msg.From, msg.Message, currentTime)
	}
}

func sendMsg(stream proto.ChitChat_ChatClient, text string, client *Client) {
	sendTime := client.incrementClock()

	msg := &proto.ClientMessage{
		LamportTimestamp: sendTime,
		From:             client.username,
		Message:          text,
	}
	if err := stream.Send(msg); err != nil {
		log.Fatalf("Event=SendError Error=%v", err)
	}

	log.Printf("Event=MessageSent LamportTime=%d Message=%s", sendTime, text)
}
