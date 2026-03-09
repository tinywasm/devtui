package devtui

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	. "github.com/tinywasm/fmt"
	"github.com/tinywasm/mcp"
)

// tabContentDTO is a Data Transfer Object for tabContent JSON
type tabContentDTO struct {
	Id             string      `json:"id"`
	Timestamp      string      `json:"timestamp"`
	Content        string      `json:"content"`
	Type           MessageType `json:"type"`
	TabTitle       string      `json:"tab_title"`
	HandlerName    string      `json:"handler_name"`
	RawHandlerName string      `json:"raw_handler_name"`
	HandlerColor   string      `json:"handler_color"`
	HandlerType    handlerType `json:"handler_type"`
	OperationID    *string     `json:"operation_id"`
	IsProgress     bool        `json:"is_progress"`
	IsComplete     bool        `json:"is_complete"`
}

// actionBaseURL strips the /logs suffix from ClientURL to get the daemon base URL.
// Used for both GET /state and POST /action requests.
func (h *DevTUI) actionBaseURL() string {
	return strings.TrimSuffix(h.ClientURL, "/logs")
}

// startSSEClient connects to the SSE endpoint and processes incoming logs
func (h *DevTUI) startSSEClient(url string) {
	// Ensure URL has protocol
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	// Fetch initial state snapshot before entering the retry loop
	h.fetchAndReconstructState()

	client := &http.Client{
		Timeout: 0, // Infinite timeout for SSE
	}

	retryDelay := 1 * time.Second

	for {
		select {
		case <-h.ExitChan:
			return
		default:
		}

		if h.Logger != nil {
			h.Logger("Connecting to SSE stream at", url)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			if h.Logger != nil {
				h.Logger("Error creating SSE request:", err)
			}
			time.Sleep(retryDelay)
			continue
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")
		if h.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+h.apiKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			if h.Logger != nil {
				h.Logger("Error connecting to SSE server:", err)
			}
			time.Sleep(retryDelay)
			continue
		}

		reader := bufio.NewReader(resp.Body)
		var currentEvent string

		// Process the stream
		for {
			select {
			case <-h.ExitChan:
				resp.Body.Close()
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if h.Logger != nil {
					h.Logger("Error reading SSE stream:", err)
				}
				break // Break inner loop to reconnect
			}

			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "event:") {
				currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}

			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if data == "" {
					continue
				}
				switch currentEvent {
				default: // "" or "log"
					h.handleLogEvent(data)
				}
				currentEvent = "" // reset after data line
			}
		}

		resp.Body.Close()
		time.Sleep(retryDelay)
	}
}

// mcpClient builds a stateless JSON-RPC client targeting the daemon's /mcp endpoint.
// ClientURL = "http://host:port/logs" → base URL = "http://host:port"
func (h *DevTUI) mcpClient() *mcp.Client {
	baseURL := strings.TrimSuffix(h.ClientURL, "/logs")
	return mcp.NewClient(baseURL, h.apiKey)
}

// handleLogEvent processes a plain log SSE data line.
func (h *DevTUI) handleLogEvent(data string) {
	var dto tabContentDTO
	if err := json.Unmarshal([]byte(data), &dto); err != nil {
		if h.Logger != nil {
			h.Logger("Error unmarshalling SSE data:", err)
		}
		return
	}

	// HandlerType 0 = TypeStateRefresh signal from daemon
	if dto.HandlerType == 0 {
		h.fetchAndReconstructState()
		h.RefreshUI()
		return
	}

	var section *tabSection
	for _, s := range h.TabSections {
		if s.title == dto.TabTitle {
			section = s
			break
		}
	}
	if section == nil {
		return
	}

	content := tabContent{
		Id:             dto.Id,
		Timestamp:      dto.Timestamp,
		Content:        dto.Content,
		Type:           dto.Type,
		tabSection:     section,
		operationID:    dto.OperationID,
		isProgress:     dto.IsProgress,
		isComplete:     dto.IsComplete,
		handlerName:    padHandlerName(dto.HandlerName, HandlerNameWidth),
		RawHandlerName: dto.HandlerName,
		handlerColor:   dto.HandlerColor,
		handlerType:    dto.HandlerType,
	}

	section.mu.Lock()
	section.tabContents = append(section.tabContents, content)
	if len(section.tabContents) > 500 {
		section.tabContents = section.tabContents[len(section.tabContents)-500:]
	}
	section.mu.Unlock()

	h.tabContentsChan <- content
}

// clearRemoteHandlers removes all fields that were added via SSE state reconstruction.
// Called before applying a new state event so stale remote handlers don't accumulate.
func (h *DevTUI) clearRemoteHandlers() {
	for _, s := range h.TabSections {
		filtered := s.fieldHandlers[:0]
		for _, f := range s.fieldHandlers {
			if !f.isRemote {
				filtered = append(filtered, f)
			}
		}
		s.fieldHandlers = filtered
	}
}

// fetchAndReconstructState fetches the daemon state snapshot and builds remote handlers via JSON-RPC.
func (h *DevTUI) fetchAndReconstructState() {
	h.mcpClient().Call(context.Background(), "tinywasm/state", map[string]string{}, func(result []byte, err error) {
		if err != nil || result == nil {
			return
		}
		var entries []StateEntry
		if err := json.Unmarshal(result, &entries); err != nil {
			return
		}
		h.clearRemoteHandlers()
		h.reconstructRemoteHandlers(entries)
	})
}

// reconstructRemoteHandlers populates sections with remote fields from state entries.
func (h *DevTUI) reconstructRemoteHandlers(entries []StateEntry) {
	client := h.mcpClient()
	for _, entry := range entries {
		var section *tabSection
		for _, s := range h.TabSections {
			if s.title == entry.TabTitle {
				section = s
				break
			}
		}
		if section == nil {
			continue
		}
		f := newRemoteField(entry, client, section, h)
		if f != nil {
			section.addFields(f)
		}
	}
}
