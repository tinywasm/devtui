package example

import (
	"strings"
	"sync"
	"time"
)

type SimpleChatHandler struct {
	mu                 sync.RWMutex // Thread-safety for all fields
	Messages           []ChatMessage
	CurrentInput       string
	WaitingForUserFlag bool
	IsProcessing       bool
	log                func(message ...any)
}

// NewSimpleChatHandler creates a new thread-safe chat handler
func NewSimpleChatHandler() *SimpleChatHandler {
	return &SimpleChatHandler{
		WaitingForUserFlag: true, // Start in waiting state
		Messages:           make([]ChatMessage, 0),
	}
}

type ChatMessage struct {
	IsUser bool
	Text   string
	Time   time.Time
}

func (h *SimpleChatHandler) Name() string { return "SimpleChat" }

func (h *SimpleChatHandler) Label() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.IsProcessing {
		return "Processing..."
	}
	if h.WaitingForUserFlag {
		return "Type message"
	}
	return "Chat (Press Enter)"
}

func (h *SimpleChatHandler) Value() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.CurrentInput
}

func (h *SimpleChatHandler) WaitingForUser() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.WaitingForUserFlag && !h.IsProcessing
}

func (h *SimpleChatHandler) SetLog(f func(message ...any)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.log = f
}

func (h *SimpleChatHandler) Change(newValue string) {
	// Display content when field selected
	if newValue == "" && !h.getWaitingForUserFlag() && !h.getIsProcessing() {
		h.mu.RLock()
		messagesCount := len(h.Messages)
		messages := make([]ChatMessage, len(h.Messages))
		copy(messages, h.Messages)
		h.mu.RUnlock()

		if h.log != nil {
			if messagesCount == 0 {
				h.log("Welcome")
			} else {
				for _, msg := range messages {
					if msg.IsUser {
						h.log("U: " + msg.Text)
					} else {
						h.log("A: " + msg.Text)
					}
				}
			}
		}
		return
	}

	// Handle user input
	if newValue != "" && strings.TrimSpace(newValue) != "" {
		userMsg := strings.TrimSpace(newValue)

		h.mu.Lock()
		h.Messages = append(h.Messages, ChatMessage{
			IsUser: true,
			Text:   userMsg,
			Time:   time.Now(),
		})

		h.WaitingForUserFlag = false
		h.IsProcessing = true
		h.CurrentInput = ""
		h.mu.Unlock()

		if h.log != nil {
			h.log("U: " + userMsg)
			h.log("Processing...")
		}

		h.generateAIResponse(userMsg)
		return
	}

	// Empty input while waiting
	if newValue == "" && h.getWaitingForUserFlag() && !h.getIsProcessing() {
		if h.log != nil {
			h.log("Type message")
		}
		return
	}
}

func (h *SimpleChatHandler) generateAIResponse(userMessage string) {
	time.Sleep(500 * time.Millisecond) // Short delay for testing

	var response string
	switch strings.ToLower(userMessage) {
	case "hello", "hi":
		response = "Hello"
	case "help":
		response = "Help available"
	case "test":
		response = "Test OK"
	default:
		response = "Response: " + userMessage
	}

	h.mu.Lock()
	h.Messages = append(h.Messages, ChatMessage{
		IsUser: false,
		Text:   response,
		Time:   time.Now(),
	})

	h.IsProcessing = false
	h.WaitingForUserFlag = true
	h.mu.Unlock()

	if h.log != nil {
		h.log("A: " + response)
	}
}

// Thread-safe helper methods
func (h *SimpleChatHandler) getWaitingForUserFlag() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.WaitingForUserFlag
}

func (h *SimpleChatHandler) getIsProcessing() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.IsProcessing
}
