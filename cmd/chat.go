package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/chzyer/readline"
)

type Chat struct {
	client       anthropic.Client
	model        string
	systemPrompt string
	verbose      bool
	quiet        bool
	noColor      bool
}

func (c Chat) oneOff(inputMessage string) {
	// Check if stdin is being piped in
	fileInfo, _ := os.Stdin.Stat()
	if fileInfo.Mode()&os.ModeCharDevice == 0 {

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

	stream := client.Messages.NewStreaming(
		context.Background(),
		anthropic.MessageNewParams{
			Model:     anthropic.Model(c.model),
			MaxTokens: int64(4096),
			System: []anthropic.TextBlockParam{
				{Text: c.systemPrompt},
			},
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(inputMessage)),
			},
		},
	)

	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			panic(err)
		}

		switch eventVariant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch deltaVariant := eventVariant.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				os.Stdout.Write([]byte(deltaVariant.Text))
			}
		}
	}

	if err := stream.Err(); err != nil {
		panic(err)
	}
}

func (c Chat) startSession() {
	messages := []anthropic.MessageParam{}

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
		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(input)))

		// Send the messages to the Anthropic API
		stream := client.Messages.NewStreaming(
			context.Background(),
			anthropic.MessageNewParams{
				Model:     anthropic.Model(c.model),
				MaxTokens: int64(4096),
				System: []anthropic.TextBlockParam{
					{Text: c.systemPrompt},
				},
				Messages: messages,
			},
		)

		// Read the response from the API
		if !c.quiet {
			os.Stdout.Write([]byte("\n\033[31mAssistant:\033[0m\n"))
		}
		assistantResponse := ""
		accMessage := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			err := accMessage.Accumulate(event)
			if err != nil {
				panic(err)
			}

			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					assistantResponse += deltaVariant.Text
					os.Stdout.Write([]byte(deltaVariant.Text))
				}
			}
		}

		if err := stream.Err(); err != nil {
			panic(err)
		}

		// Add assistant response to conversation history
		messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(assistantResponse)))

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
