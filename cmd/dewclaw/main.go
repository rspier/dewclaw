package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rspier/dewclaw"
)

func main() {
	sysPrompt := flag.String("s", "", "System prompt")
	userPrompt := flag.String("t", "", "User prompt/text")
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

	apiKey, err := dewclaw.GetAPIKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if *userPrompt == "" {
		fmt.Fprintln(os.Stderr, "Error: user text/prompt is required (-t)")
		os.Exit(1)
	}

	client := dewclaw.NewClient(apiKey)
	msg, usage, err := client.GenerateContent(*sysPrompt, *userPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating content: %v\n", err)
		os.Exit(1)
	}

	if *verbose && usage != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] Tokens used - Prompt: %d, Response: %d, Total: %d\n",
			usage.PromptTokenCount, usage.CandidatesTokenCount, usage.TotalTokenCount)
	}

	fmt.Println(msg)
}
