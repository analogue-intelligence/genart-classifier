package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/joho/godotenv"
)

var systemDefinition = "You are an expert evaluator of software-based artworks."

func main() {
	godotenv.Load()

	ctx := context.Background()

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Println("Warning: NEBULA_API_KEY environment variable is missing")
	}
	apiEndpoint := os.Getenv("API_ENDPOINT")
	if apiEndpoint == "" {
		log.Println("Warning: API_ENDPOINT environment variable is missing")
	}

	nebulaProvider := &compat_oai.OpenAICompatible{
		Provider: "nebula",
		BaseURL:  apiEndpoint,
		APIKey:   apiKey,
	}

	g := genkit.Init(ctx, genkit.WithPlugins(nebulaProvider))

	outputSchema := map[string]any{
		"material_and_processes": "",
		"interaction":            "",
		"outcome":                "",
		"explanation":            "",
		"reuse_algorithm":        "",
	}

	resp, err := genkit.Generate(ctx, g,
		ai.WithModelName("nebula/deepseek-r1:1.5b"),
		ai.WithSystem(systemDefinition),
		ai.WithPrompt("What is the capital of Brazil?"),
		ai.WithOutputSchema(outputSchema),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response: %s\n\n", resp.Text())
}
