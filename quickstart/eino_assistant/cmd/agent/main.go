package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"github.com/cloudwego/eino-examples/agent"
	"github.com/cloudwego/eino-ext/devops"
)

func init() {
	if os.Getenv("EINO_DEBUG") == "true" {
		err := devops.Init(context.Background())
		if err != nil {
			log.Printf("[eino dev] init failed, err=%v", err)
		}
	}
}

var id = flag.String("id", "", "conversation id")

func main() {
	flag.Parse()

	if *id == "" {
		*id = strconv.Itoa(rand.IntN(1000000))
	}

	ctx := context.Background()

	err := agent.Init()
	if err != nil {
		log.Printf("[eino agent] init failed, err=%v", err)
		return
	}

	// Start interactive dialogue
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("🧑‍ : ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}

		input = strings.TrimSpace(input)
		if input == "" || input == "exit" || input == "quit" {
			return
		}

		// Call RunAgent with the input
		sr, err := agent.RunAgent(ctx, *id, input)
		if err != nil {
			fmt.Printf("Error from RunAgent: %v\n", err)
			continue
		}

		// Print the response
		fmt.Print("🤖 : ")
		for {
			msg, err := sr.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Printf("Error receiving message: %v\n", err)
				break
			}
			fmt.Print(msg.Content)
		}
		fmt.Println()
		fmt.Println()
	}
}
