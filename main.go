package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	geminiModel = "gemini-2.5-flash"
	apiURL      = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
)

const systemPrompt = `You are a professional software engineer. 
Write a concise, conventional commit message based on the provided diff.
Only output the commit message, no markdown formatting,
no explanations.
When appropriate, output a Key Features section as a bulleted list.

`

var (
	verbose      = flag.Bool("v", false, "verbose output")
	contextLines = flag.Int("U", 1, "lines of context in diff")
)

func main() {
	flag.Parse()
	apiKey := os.Getenv("GEMINI_API_KEY")

	configPath := ""
	if configDir, err := os.UserConfigDir(); err == nil {
		configPath = filepath.Join(configDir, "quickdesc", "api_key")
		if apiKey == "" {
			if content, err := os.ReadFile(configPath); err == nil {
				apiKey = strings.TrimSpace(string(content))
			}
		}
	}

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GEMINI_API_KEY environment variable is not set.")
		if configPath != "" {
			fmt.Fprintf(os.Stderr, "Alternatively, you can place your raw API key in a file at:\n  %s\n", configPath)
		}
		os.Exit(1)
	}

	diffText := getDiff()
	if strings.TrimSpace(diffText) == "" {
		fmt.Fprintln(os.Stderr, "No changes detected. Stage changes in Git or use JJ.")
		os.Exit(1)
	}

	// Optimize context length to maximize free tier token limits
	// 50,000 chars is roughly 12,000 tokens. The model has a 1M token window,
	// but reducing input size keeps TPM quota usage low.
	if len(diffText) > 50000 {
		if *verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] Truncating diff from %d to 50000 chars\n", len(diffText))
		}
		diffText = diffText[:50000] + "\n...[diff truncated for length constraints]"
	}

	startTime := time.Now()
	msg, usage, err := generateCommitMessage(apiKey, diffText)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating message: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		duration := time.Since(startTime)
		fmt.Fprintf(os.Stderr, "[DEBUG] API call took %v\n", duration)
		if usage != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Tokens used - Prompt: %d, Response: %d, Total: %d\n",
				usage.PromptTokenCount, usage.CandidatesTokenCount, usage.TotalTokenCount)
		}
	}

	fmt.Println(msg)
}

func getDiff() string {
	unifiedArg := fmt.Sprintf("-U%d", *contextLines)
	jjContextArg := fmt.Sprintf("%d", *contextLines)

	// 1. Try Git cached (staged changes) with context flag
	cmd := exec.Command("git", "diff", "--cached", unifiedArg)
	out, err := cmd.CombinedOutput()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		return string(out)
	}

	// 2. Try regular Git diff with context flag
	cmd = exec.Command("git", "diff", unifiedArg)
	out, err = cmd.CombinedOutput()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		return string(out)
	}

	// 3. Try JJ (Jujutsu) diff with context flag
	cmd = exec.Command("jj", "diff", "--git", "--context", jjContextArg)
	out, err = cmd.CombinedOutput()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		return string(out)
	}

	return ""
}

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

func generateCommitMessage(apiKey, diff string) (string, *UsageMetadata, error) {
	reqBody := GenerateContentRequest{
		SystemInstruction: &Content{
			Parts: []Part{
				{Text: systemPrompt},
			},
		},
		Contents: []Content{
			{
				Parts: []Part{
					{Text: diff},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, err
	}

	url := fmt.Sprintf(apiURL, geminiModel, apiKey)
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
