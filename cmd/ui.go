package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	newText   = lipgloss.AdaptiveColor{Light: "#FF0088", Dark: "#FF69B4"}

	dividerStyle = lipgloss.NewStyle().
			Foreground(subtle).
			MarginLeft(2).
			MarginRight(2)

	dividerChar = "─"

	userStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(special).
			PaddingLeft(2)

	assistantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			PaddingLeft(2)

	inputBoxStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	viewportStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2)

	newTextStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(newText).
			PaddingLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(special).
			PaddingLeft(2)
)

type chatModel struct {
	viewport           viewport.Model
	messages           []string
	textarea           textarea.Model
	err                error
	chat               *Chat
	streaming          bool
	currentMsg         string
	stream             *openai.ChatCompletionStream
	width              int
	height             int
	colorTrail         []string
	showingModelSelect bool
	modelSelector      list.Model
}

type modelSelectModel struct {
	list     list.Model
	choice   string
	quitting bool
}

type modelItem struct {
	name, desc string
}

func (i modelItem) Title() string       { return i.name }
func (i modelItem) Description() string { return i.desc }
func (i modelItem) FilterValue() string { return i.name }

func getAvailableModels(client *openai.Client) ([]list.Item, error) {
	models, err := client.ListModels(context.Background())
	if err != nil {
		return nil, err
	}

	// Filter and sort GPT models
	var items []list.Item
	for _, model := range models.Models {
		// Only include GPT-3.5 and GPT-4 models
		if strings.Contains(model.ID, "gpt-3.5") || strings.Contains(model.ID, "gpt-4") {
			desc := "GPT model"
			if strings.Contains(model.ID, "gpt-4") {
				desc = "Most capable model, best for complex tasks"
			} else if strings.Contains(model.ID, "gpt-3.5") {
				desc = "Faster and cheaper, good for simpler tasks"
			}
			items = append(items, modelItem{
				name: model.ID,
				desc: desc,
			})
		}
	}

	// Sort models so GPT-4 appears first
	sort.Slice(items, func(i, j int) bool {
		return strings.Contains(items[i].(modelItem).name, "gpt-4")
	})

	return items, nil
}

func initialModel(chat *Chat) chatModel {
	// Clear the terminal first
	fmt.Print("\033[H\033[2J")  // ANSI escape codes to clear screen and move cursor to home position
	
	ta := textarea.New()
	ta.Placeholder = fmt.Sprintf("Using %s (type /model to change or /help for help) - Tab for newline", chat.model)
	ta.Focus()
	ta.Prompt = "│ "
	ta.CharLimit = 4000
	ta.SetWidth(30)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(0, 0)
	vp.Style = viewportStyle

	var items []list.Item
	apiItems, err := getAvailableModels(chat.client)
	if err != nil {
		// Fallback to default models if API call fails
		items = []list.Item{
			modelItem{"gpt-4", "Most capable model, best at complex tasks"},
			modelItem{"gpt-4-turbo-preview", "Latest GPT-4 model with lower cost"},
			modelItem{"gpt-3.5-turbo", "Faster and cheaper, good for simpler tasks"},
		}
	} else {
		items = apiItems
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select AI Model (Enter to choose, Esc to cancel)"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		MarginLeft(2).
		Foreground(highlight)

	m := chatModel{
		textarea:      ta,
		messages:      []string{},
		viewport:      vp,
		chat:          chat,
		streaming:     false,
		stream:        nil,
		colorTrail:    make([]string, 0),
		modelSelector: l,
	}

	// Add initial system message
	m.messages = append(m.messages, 
		assistantStyle.Render("System: ") + fmt.Sprintf("Using %s (type /help for commands)", chat.model))
	m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))

	return m
}

func initialModelSelect() modelSelectModel {
	items := []list.Item{
		modelItem{"gpt-4", "Most capable model, best at complex tasks"},
		modelItem{"gpt-4-turbo-preview", "Latest GPT-4 model with lower cost"},
		modelItem{"gpt-3.5-turbo", "Faster and cheaper, good for simpler tasks"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select AI Model"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		MarginLeft(2).
		Foreground(highlight)

	return modelSelectModel{list: l}
}

func (m chatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m modelSelectModel) Init() tea.Cmd {
	return nil
}

func renderMessage(renderer *glamour.TermRenderer, style lipgloss.Style, prefix, content string, width int) string {
	// Render markdown content
	rendered, err := renderer.Render(content)
	if err != nil {
		rendered = content
	}
	// Remove trailing newline from markdown render
	rendered = strings.TrimSpace(rendered)

	// Add prefix with style
	return style.Render(prefix) + rendered
}

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd   tea.Cmd
		vpCmd   tea.Cmd
		listCmd tea.Cmd
	)

	if m.showingModelSelect {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc:
				m.showingModelSelect = false
				return m, nil
			case tea.KeyEnter:
				if i, ok := m.modelSelector.SelectedItem().(modelItem); ok {
					m.chat.model = i.name
					m.messages = append(m.messages,
						assistantStyle.Render("System: ")+fmt.Sprintf("Switched to %s", i.name))
					m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))
					m.textarea.Placeholder = fmt.Sprintf("Using %s (type /model to change or /help for help) - Tab for newline", i.name)
				}
				m.showingModelSelect = false
				return m, nil
			}
		}

		// Set the model selector size to use most of the window
		m.modelSelector.SetSize(m.width-4, m.height-4)

		m.modelSelector, listCmd = m.modelSelector.Update(msg)
		return m, listCmd
	}

	// Update textarea and viewport
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.stream != nil {
				m.stream.Close()
			}
			return m, tea.Quit

		case tea.KeyTab:
			m.textarea.SetValue(m.textarea.Value() + "\n")
			return m, nil

		case tea.KeyEnter:
			input := m.textarea.Value()

			// Handle commands
			switch strings.TrimSpace(input) {
			case "/help":
				helpText := `Available commands:
• /help  - Show this help message
• /model - Change the AI model
• /clear - Clear conversation history
• TAB    - Insert newline`
				m.messages = append(m.messages, helpStyle.Render(helpText))
				m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))
				m.textarea.Reset()
				return m, nil

			case "/model":
				m.showingModelSelect = true
				m.textarea.Reset()
				return m, nil

			case "/clear":
				// Clear messages but keep the model info
				m.chat.messages = []openai.ChatCompletionMessage{}
				m.messages = []string{assistantStyle.Render("System: ") + "Conversation history cleared"}
				m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))
				m.textarea.Reset()
				return m, nil
			}

			if input != "" {
				m.messages = append(m.messages,
					userStyle.Render("You: ")+input)
				m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))
				m.textarea.Reset()
				m.viewport.GotoBottom()

				stream, err := m.chat.streamResponse(input)
				if err != nil {
					m.messages = append(m.messages,
						assistantStyle.Render("Error: ")+err.Error())
					m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))
					return m, nil
				}

				m.streaming = true
				m.stream = stream
				return m, m.chat.receiveStreamResponse(stream)
			}
		}

	case streamMsg:
		if m.streaming {
			if msg.done {
				m.streaming = false
				m.chat.messages = append(m.chat.messages, openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: m.currentMsg,
				})
				m.messages = append(m.messages,
					assistantStyle.Render("Assistant: ")+m.currentMsg)
				m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))
				m.viewport.GotoBottom()
				m.currentMsg = ""
				m.stream = nil
				m.colorTrail = make([]string, 0)
				return m, nil
			}

			m.currentMsg += msg.content

			// Update color trail
			m.colorTrail = append(m.colorTrail, msg.content)
			if len(m.colorTrail) > 8 { // Keep last 8 chunks for color trail
				m.colorTrail = m.colorTrail[1:]
			}

			// Build the current message with color trail
			var displayMsg strings.Builder
			displayMsg.WriteString(assistantStyle.Render("Assistant: "))
			displayMsg.WriteString(m.currentMsg[:len(m.currentMsg)-len(strings.Join(m.colorTrail, ""))])

			// Add color trail with gradually fading colors
			for i, chunk := range m.colorTrail[:len(m.colorTrail)-1] {
				fade := float64(i) / float64(len(m.colorTrail)-1)
				color := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x",
					int(255-(fade*167)), // R: 255 -> 88
					int(0+(fade*102)),   // G: 0 -> 102
					int(136+(fade*8)),   // B: 136 -> 144
				))
				displayMsg.WriteString(lipgloss.NewStyle().Foreground(color).Render(chunk))
			}

			// Add newest chunk in bright pink
			if len(m.colorTrail) > 0 {
				displayMsg.WriteString(newTextStyle.Render(m.colorTrail[len(m.colorTrail)-1]))
			}

			currentDisplay := append(m.messages, displayMsg.String())
			content := strings.Join(currentDisplay, "\n"+dividerStyle.Render(dividerChar)+"\n")
			m.viewport.SetContent(content)
			m.viewport.GotoBottom()

			return m, m.chat.receiveStreamResponse(m.stream)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.showingModelSelect {
			m.modelSelector.SetSize(msg.Width-4, msg.Height-4)
			return m, nil
		}

		// Update these dimensions
		headerHeight := 0     // No header
		footerHeight := 4     // Space for input area (reduced from 6)
		verticalMarginHeight := 0  // Remove margin height (was 2)

		m.textarea.SetWidth(msg.Width - 4)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - footerHeight - verticalMarginHeight - headerHeight

		if m.viewport.Height < 0 {
			m.viewport.Height = 0
		}

		m.viewport.SetContent(strings.Join(m.messages, "\n"+dividerStyle.Render(dividerChar)+"\n"))
		m.viewport.GotoBottom()
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m modelSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(modelItem)
			if ok {
				m.choice = i.name
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m chatModel) View() string {
	if m.showingModelSelect {
		return m.modelSelector.View()
	}

	dividerLine := strings.Repeat(dividerChar, m.width/2)
	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.viewport.View(),
		dividerStyle.Render(dividerLine),
		m.textarea.View(),
	)
}

func (m modelSelectModel) View() string {
	return m.list.View()
}
