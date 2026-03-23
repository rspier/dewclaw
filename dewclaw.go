package dewclaw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	geminiModel = "gemini-3-flash-preview"
	apiURL      = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
)

type Part struct {
	Text string `json:"text"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type GenerateContentRequest struct {
	SystemInstruction *Content  `json:"system_instruction,omitempty"`
	Contents          []Content `json:"contents"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type GenerateContentResponse struct {
	Candidates []struct {
		Content Content `json:"content"`
	} `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

type Client struct {
	APIKey string
}

func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey}
}

func (c *Client) GenerateContent(systemPrompt, userPrompt string) (string, *UsageMetadata, error) {
	reqBody := GenerateContentRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: userPrompt},
				},
			},
		},
	}

	if systemPrompt != "" {
		reqBody.SystemInstruction = &Content{
			Parts: []Part{
				{Text: systemPrompt},
			},
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, err
	}

	url := fmt.Sprintf(apiURL, geminiModel, c.APIKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp GenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", nil, err
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return "", nil, fmt.Errorf("no content returned from API")
	}

	return strings.TrimSpace(apiResp.Candidates[0].Content.Parts[0].Text), &apiResp.UsageMetadata, nil
}

// GetAPIKey resolves the Gemini API key from the GEMINI_API_KEY environment variable
// or from the user's config directory (~/.config/dewclaw/api_key).
func GetAPIKey() (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey != "" {
		return apiKey, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "dewclaw", "api_key")
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("GEMINI_API_KEY not set and config file not found at %s", configPath)
		}
		return "", fmt.Errorf("error reading config file at %s: %v", configPath, err)
	}

	return strings.TrimSpace(string(content)), nil
}
