package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var id = flag.String("id", "", "conversation id")

func main() {
	flag.Parse()

	if *id == "" {
		*id = strconv.Itoa(rand.IntN(1000000))
	}

	// 开始交互式对话
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

		sendMessage(*id, input)
	}
}

func sendMessage(id, message string) {
	baseURL := "http://127.0.0.1:8888/chat"
	params := url.Values{}
	params.Add("id", id)
	params.Add("msg", message)
	reqURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(reqURL)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Print("🤖 : ")
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			content := strings.TrimPrefix(line, "data:")
			content = strings.TrimSpace(content)
			if content != "" {
				fmt.Print(content)
			}
		}
	}
	fmt.Println()
	fmt.Println()
}
