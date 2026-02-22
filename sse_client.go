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

// startSSEClient connects to the SSE endpoint and processes incoming logs
func (h *DevTUI) startSSEClient(url string) {
	// Ensure URL has protocol
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	client := &http.Client{
		Timeout: 0, // Infinite timeout for SSE
	}

	retryDelay := 1 * time.Second

	for {
		// Only run if channel is open (app is running)
		// Note: reading from closed channel returns immediately, so checking if h.ExitChan is closed is tricky without blocking
		// Instead we check if the context is done if we had one, or rely on read error when app closes
		// But here we can check non-blocking read
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

		// Process the stream
		for {
			// Check exit signal periodically or on error
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
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimSpace(data)

				if data == "" {
					continue
				}

				var dto tabContentDTO
				if err := json.Unmarshal([]byte(data), &dto); err != nil {
					if h.Logger != nil {
						h.Logger("Error unmarshalling SSE data:", err)
					}
					continue
				}

				// Find local section by title
				var section *tabSection
				for _, s := range h.TabSections {
					if s.title == dto.TabTitle {
						section = s
						break
					}
				}

				if section == nil {
					continue
				}

				// Create internal content structure
				content := tabContent{
					Id:             dto.Id,
					Timestamp:      dto.Timestamp,
					Content:        dto.Content,
					Type:           dto.Type,
					tabSection:     section,
					operationID:    dto.OperationID,
					isProgress:     dto.IsProgress,
					isComplete:     dto.IsComplete,
					handlerName:    dto.HandlerName,
					RawHandlerName: dto.RawHandlerName,
					handlerColor:   dto.HandlerColor,
					handlerType:    dto.HandlerType,
				}

				// Send to main loop
				h.tabContentsChan <- content
			}
		}

		resp.Body.Close()
		time.Sleep(retryDelay)
	}
}
