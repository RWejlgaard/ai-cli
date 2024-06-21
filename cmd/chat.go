package cmd

import (
	"bufio"
	"fmt"
	"os"
	"context"
	openai "github.com/sashabaranov/go-openai"
	"errors"
	"io"
)

func oneOff(client *openai.Client, model string, systemPrompt string, inputMessage string, verbose bool) {
	// Check if stdin is being piped in
	fileInfo, err := os.Stdin.Stat()
	if fileInfo.Mode() & os.ModeCharDevice == 0 {

		// Read from stdin
		stdinString := ""
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			stdinString = fmt.Sprintf("%s\n%s", stdinString, scanner.Text())
		}
		// remove the first newline
		stdinString = stdinString[1:]

		inputMessage = fmt.Sprintf("%s\n\n```\n%s\n```", inputMessage, stdinString)
	}

	if verbose {
		println("USER INPUT: ", inputMessage)
	}

	resp, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: "system",
					Content: systemPrompt,
				},
				{
					Role: "user",
					Content: inputMessage,
				},
			},
			Stream: true,
		},
	)
	defer resp.Close()

	if err != nil {
		panic(err)
	}

	for {
		msg, err := resp.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}

		if msg.Choices[0].Delta.Content != "" {
			print(msg.Choices[0].Delta.Content)
		}
	}
}

func startSession(client *openai.Client, model string, systemPrompt string, verbose bool) {
	messages := []openai.ChatCompletionMessage{
		{
			Role: "system",
			Content: systemPrompt,
		},
	}

	var input string
	
	for {
		input = ""
		// Read input from the user
		fmt.Print("\033[32mYou:\033[0m ")
		
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input = scanner.Text()

		if verbose {
			println("USER INPUT: ", input)
		}

		// Add the user's message to the list of messages
		messages = append(messages, openai.ChatCompletionMessage{
			Role: "user",
			Content: input,
		})

		// Send the messages to the OpenAI API
		resp, err := client.CreateChatCompletionStream(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: model,
				Messages: messages,
				Stream: true,
			},
		)
		
		if err != nil {
			panic(err)
		}

		// Read the response from the API

		println()
		fmt.Println("\033[31mAssistant:\033[0m")
		assistantResponse := ""
		for {
			msg, err := resp.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					// fmt.Println("EOF!!!!")
					messages = append(messages, openai.ChatCompletionMessage{
						Role: "system",
						Content: assistantResponse,
					})
					resp.Close()
					break
				}
				panic(err)
			}
			if msg.Choices[0].Delta.Content != "" {
				assistantResponse = fmt.Sprintf("%s%s", assistantResponse, msg.Choices[0].Delta.Content)
				print(msg.Choices[0].Delta.Content)
			}
		}
		println()
		println()

	}
}