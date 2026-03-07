package devtui

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	. "github.com/tinywasm/fmt"
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
	h.fetchAndReconstructState(h.actionBaseURL())

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
				case "state":
					h.handleStateEvent(data)
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

// handleLogEvent processes a plain log SSE data line.
func (h *DevTUI) handleLogEvent(data string) {
	var dto tabContentDTO
	if err := json.Unmarshal([]byte(data), &dto); err != nil {
		if h.Logger != nil {
			h.Logger("Error unmarshalling SSE data:", err)
		}
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

// handleStateEvent processes a live "event: state" SSE data line.
// It replaces any previously reconstructed remote handlers with fresh ones from the payload.
func (h *DevTUI) handleStateEvent(data string) {
	var entries []StateEntry
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		return
	}
	h.clearRemoteHandlers()
	h.reconstructRemoteHandlers(entries)
	h.RefreshUI()
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

// fetchAndReconstructState fetches the daemon state snapshot and builds remote handlers.
// Degrades gracefully if /state is unavailable.
func (h *DevTUI) fetchAndReconstructState(baseURL string) {
	resp, err := http.Get(baseURL + "/state")
	if err != nil || resp.StatusCode != 200 {
		return
	}
	defer resp.Body.Close()
	var entries []StateEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return
	}
	h.clearRemoteHandlers()
	h.reconstructRemoteHandlers(entries)
}

// reconstructRemoteHandlers populates sections with remote fields from state entries.
func (h *DevTUI) reconstructRemoteHandlers(entries []StateEntry) {
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
		f := newRemoteField(entry, h.actionBaseURL(), section)
		if f != nil {
			section.addFields(f)
		}
	}
}
