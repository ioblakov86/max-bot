package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// OpenRouterConfig holds the configuration for OpenRouter API
type OpenRouterConfig struct {
	APIKey string
	Model  string
	Prompt string
}

// MessageAnalysis represents the structure of analyzed message
type MessageAnalysis struct {
	OriginalMessage string `json:"original_message"`
	IsValid         bool   `json:"is_valid"`
	Employee        struct {
		Position  string `json:"position"`
		FullName  string `json:"full_name"`
	} `json:"employee"`
	AbsenceType string `json:"absence_type"`
	Dates       struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:"dates"`
	Status     string `json:"status"`
	Substitute string `json:"substitute"`
}

// OpenRouterRequest represents the request structure for OpenRouter API
type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a message in the OpenRouter request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response structure from OpenRouter API
type OpenRouterResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOpenRouterConfig creates a new configuration for OpenRouter API
func NewOpenRouterConfig(prompt string) *OpenRouterConfig {
	return &OpenRouterConfig{
		APIKey: os.Getenv("OPENROUTER_API_KEY"),
		Model:  getFreeModel(),
		Prompt: prompt,
	}
}

// getFreeModel returns one of the free models available
func getFreeModel() string {
	model := os.Getenv("OPENROUTER_MODEL")
	if model != "" {
		return model
	}

	// Default to one of the free models
	return "arcee-ai/trinity-large-preview:free"
}

// AnalyzeMessage sends a message to OpenRouter API for analysis
// Returns an array of MessageAnalysis (can contain multiple employees)
func (c *OpenRouterConfig) AnalyzeMessage(messageText string) ([]MessageAnalysis, error) {
	if c.APIKey == "" {
		return []MessageAnalysis{
			{
				OriginalMessage: messageText,
				IsValid:         false,
			},
		}, fmt.Errorf("OPENROUTER_API_KEY is not set in environment variables")
	}

	// Prepare the full prompt with the message to analyze
	fullPrompt := fmt.Sprintf("%s\n\nСообщение: %s", c.Prompt, messageText)

	request := OpenRouterRequest{
		Model: c.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: fullPrompt,
			},
		},
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Check if the response status indicates an error
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openRouterResp OpenRouterResponse
	err = json.Unmarshal(body, &openRouterResp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Extract the content from the response
	if len(openRouterResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned in the response")
	}

	content := openRouterResp.Choices[0].Message.Content

	// Find the JSON array in the response (it should be between square brackets)
	startIdx := strings.Index(content, "[")
	endIdx := strings.LastIndex(content, "]")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return nil, fmt.Errorf("no valid JSON array found in response: %s", content)
	}

	jsonStr := content[startIdx : endIdx+1]

	// Parse the JSON response into array of MessageAnalysis
	var analyses []MessageAnalysis
	err = json.Unmarshal([]byte(jsonStr), &analyses)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling analysis results: %w", err)
	}

	// Ensure the original message is preserved in each analysis
	for i := range analyses {
		analyses[i].OriginalMessage = messageText
	}

	return analyses, nil
}