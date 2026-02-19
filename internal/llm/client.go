// Package llm provides a client for GitHub Models (chat completions API).
// It supports both streaming and non-streaming modes, using the X-GitHub-Token
// forwarded from Copilot Extension requests for authentication.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Role constants for chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request body for the chat completions API.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

// ChatResponse is the response from the chat completions API.
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
		Delta   struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Client provides access to the GitHub Models chat completions API.
type Client struct {
	endpoint  string
	model     string
	maxTokens int
	timeout   time.Duration
	client    *http.Client
}

// NewClient creates a new LLM client.
func NewClient(endpoint, model string, maxTokens int, timeout time.Duration) *Client {
	return &Client{
		endpoint:  endpoint,
		model:     model,
		maxTokens: maxTokens,
		timeout:   timeout,
		client:    &http.Client{Timeout: timeout},
	}
}

// Complete performs a non-streaming chat completion.
func (c *Client) Complete(ctx context.Context, token, systemPrompt string, messages []ChatMessage) (string, error) {
	allMessages := make([]ChatMessage, 0, len(messages)+1)
	if systemPrompt != "" {
		allMessages = append(allMessages, ChatMessage{Role: RoleSystem, Content: systemPrompt})
	}
	allMessages = append(allMessages, messages...)

	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    allMessages,
		MaxTokens:   c.maxTokens,
		Temperature: 0.3,
		Stream:      false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.endpoint + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Stream performs a streaming chat completion. Returns channels for content chunks and errors.
// The caller MUST drain the content channel to avoid goroutine leaks. The error channel
// will receive at most one error and is closed when the stream completes.
func (c *Client) Stream(ctx context.Context, token, systemPrompt string, messages []ChatMessage) (<-chan string, <-chan error) {
	contentCh := make(chan string, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(contentCh)
		defer close(errCh)

		allMessages := make([]ChatMessage, 0, len(messages)+1)
		if systemPrompt != "" {
			allMessages = append(allMessages, ChatMessage{Role: RoleSystem, Content: systemPrompt})
		}
		allMessages = append(allMessages, messages...)

		reqBody := ChatRequest{
			Model:       c.model,
			Messages:    allMessages,
			MaxTokens:   c.maxTokens,
			Temperature: 0.3,
			Stream:      true,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errCh <- fmt.Errorf("marshal request: %w", err)
			return
		}

		url := c.endpoint + "/chat/completions"
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("create request: %w", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.client.Do(req)
		if err != nil {
			errCh <- fmt.Errorf("request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			// Check context before processing each line to allow early exit
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var chatResp ChatResponse
			if err := json.Unmarshal([]byte(data), &chatResp); err != nil {
				continue
			}

			if len(chatResp.Choices) > 0 && chatResp.Choices[0].Delta.Content != "" {
				select {
				case contentCh <- chatResp.Choices[0].Delta.Content:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("stream read: %w", err)
		}
	}()

	return contentCh, errCh
}
