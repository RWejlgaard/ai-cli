package cmd

import (
	"context"
	"fmt"
	"os"
	openai "github.com/sashabaranov/go-openai"
	tea "github.com/charmbracelet/bubbletea"
)

type Chat struct {
	client       *openai.Client
	model        string
	systemPrompt string
	verbose      bool
	quiet        bool
	noColor      bool
	messages     []openai.ChatCompletionMessage
}

// New message type for streaming responses
type streamMsg struct {
	content string
	done    bool
}

func (c *Chat) streamResponse(inputMessage string) (*openai.ChatCompletionStream, error) {
	if c.verbose {
		println("USER INPUT: ", inputMessage)
	}

	c.messages = append(c.messages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: inputMessage,
	})

	stream, err := c.client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    c.model,
			Messages: c.messages,
			Stream:   true,
		},
	)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

type streamResponseMsg struct {
	stream *openai.ChatCompletionStream
	err    error
}

func (c *Chat) receiveStreamResponse(stream *openai.ChatCompletionStream) tea.Cmd {
	return func() tea.Msg {
		response, err := stream.Recv()
		if err != nil {
			stream.Close()
			return streamMsg{content: "", done: true}
		}
		return streamMsg{content: response.Choices[0].Delta.Content, done: false}
	}
}

func (c *Chat) getResponse(inputMessage string) string {
	if c.verbose {
		println("USER INPUT: ", inputMessage)
	}

	c.messages = append(c.messages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: inputMessage,
	})

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    c.model,
			Messages: c.messages,
		},
	)

	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	c.messages = append(c.messages, openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: resp.Choices[0].Message.Content,
	})

	return resp.Choices[0].Message.Content
}

func (c *Chat) startSession() {
	// Initialize chat history with system prompt
	c.messages = []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: c.systemPrompt,
		},
	}

	// Start the chat UI
	p := tea.NewProgram(initialModel(c))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}

func (c *Chat) UpdateAPIKey(apiKey string) {
	c.client = openai.NewClient(apiKey)
}