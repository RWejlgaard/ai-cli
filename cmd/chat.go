package cmd

import (
	"bufio"
	"fmt"
	"os"
	"context"
	openai "github.com/sashabaranov/go-openai"
	"errors"
	"github.com/chzyer/readline"
	"io"
)

type Chat struct {
	client *openai.Client
	model string
	systemPrompt string
	verbose bool
	quiet bool
	noColor bool
}

func (c Chat) oneOff(inputMessage string) {
	// Check if stdin is being piped in
	fileInfo, _ := os.Stdin.Stat()
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

	if c.verbose {
		println("USER INPUT: ", inputMessage)
	}

	resp, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: "system",
					Content: c.systemPrompt,
				},
				{
					Role: "user",
					Content: inputMessage,
				},
			},
			Stream: true,
		},
	)

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
			os.Stdout.Write([]byte(msg.Choices[0].Delta.Content))
		}
	}
	resp.Close()
}

func (c Chat) startSession() {
	messages := []openai.ChatCompletionMessage{
		{
			Role: "system",
			Content: c.systemPrompt,
		},
	}
	
	for {
		// Get input from the user
		prompt := "You: "
		if c.quiet {
			prompt = ""
		}
		input, err := getInputFromUser(prompt, c.noColor)
		if err != nil {
			panic(err)
		}

		if c.verbose {
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
				Model: c.model,
				Messages: messages,
				Stream: true,
			},
		)
		
		if err != nil {
			panic(err)
		}

		// Read the response from the API

		if !c.quiet {
			os.Stdout.Write([]byte("\n\033[31mAssistant:\033[0m\n"))
		}
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
				os.Stdout.Write([]byte(msg.Choices[0].Delta.Content))
			}
		}

		os.Stdout.Write([]byte("\n"))
		if !c.quiet {
			os.Stdout.Write([]byte("\n"))
		}

	}
}

func getInputFromUser(prompt string, colorDisabled bool) (string, error) {
	// Create a readline instance
	if !colorDisabled {
		prompt = fmt.Sprintf("\033[32m%s\033[0m", prompt)
	}

	rl, err := readline.New(prompt)
	if err != nil {
		return "", err
	}
	defer rl.Close()

	// Read input from the user
	line, err := rl.Readline()
	if err == readline.ErrInterrupt {
		os.Exit(0)
	}
	if err != nil {
		return "", err
	}

	return line, nil
}