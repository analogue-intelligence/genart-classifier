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

	"bytes"
	_ "embed"
	"text/template"
)

//go:embed prompts/baseline/original_prompt.txt
var baselineTemplate string

//go:embed prompts/icl/original_prompt_few.txt
var baselineFewTemplate string

var exampleProgram = `
function setup() {
  createCanvas(400, 400);
}
function draw() {
  background(220);
  let d = dist(mouseX, mouseY, width/2, height/2);
  fill(d, 100, 255);
  ellipse(width/2, height/2, d);
}
`

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
		ai.WithModelName("nebula/deepseek-r1:8b"),
		ai.WithSystem(systemDefinition),
		ai.WithPrompt(ExecutePrompt(exampleProgram)),
		ai.WithOutputSchema(outputSchema),
		ai.WithConfig(map[string]any{
			"Temperature": 0.0,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response: %s\n\n", resp.Text())
}

func ExecutePrompt(artwork string) string {
	tmpl, _ := template.New("prompt").Parse(baselineTemplate)

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Artwork":   artwork,
		"Extension": ".js",
		"File_name": "bla",
	}
	tmpl.Execute(&buf, data)
	return buf.String()
}
