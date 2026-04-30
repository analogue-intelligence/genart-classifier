package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"

	// "github.com/firebase/genkit/go/plugins/firebase"

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

var MODEL_NAME = "nebula/deepseek-r1:8b"

func main() {
	fmt.Println("Runing Script...")
	ctx, g := setupLLMConfiguration()

	outputSchema := map[string]any{
		"material_and_processes": "",
		"interaction":            "",
		"outcome":                "",
		"explanation":            "",
		"reuse_algorithm":        "",
	}

	artworks_path := getArtworksPath()

	readFolder(artworks_path, ctx, g, outputSchema)
}

func readFolder(artworks_path string, ctx context.Context, g *genkit.Genkit, outputSchema map[string]any) {
	err := filepath.WalkDir(artworks_path, func(filep string, info os.DirEntry, err error) error {
		if !info.IsDir() && filepath.Ext(filep) == ".js" {
			fmt.Println("Reading artwork: " + info.Name())

			artworkSourceCode, err := os.ReadFile(filep)
			// var response *ai.ModelResponse
			if err != nil {
				panic("Reading the content of the file doesn't work.")
			}
			response, errs := queryLLM(ctx, g, outputSchema, string(artworkSourceCode))

			if errs != nil {
				// try one more time
				response, _ = queryLLM(ctx, g, outputSchema, string(artworkSourceCode))
			}
			processResponse(response, info.Name())
		}
		return nil
	})
	if err != nil {
		panic("Error while walking the directory")
	}
}

func processResponse(response *ai.ModelResponse, filename string) {
	response_path := createFile(filename)

	if err := os.WriteFile(response_path, []byte(response.Text()), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("Processed file: %v \n", filename)
}

func createFile(filename string) string {
	rootPath, err := getRootPath()
	if err != nil {
		panic("The root path does not exist")
	}
	outputPath := filepath.Join(rootPath, "LLM-output-"+MODEL_NAME)

	// Owner (7) = 4 (read) + 2 (write) + 1 (execute): rwx
	// Group: r-x (5) =  4 (read) + 0 (no write) + 1 (execute)
	// Others: r-x (5) =  4 (read) + 0 (no write) + 1 (execute)
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		panic(err)
	}

	response_path := filepath.Join(outputPath, filename)

	return response_path
}

func getArtworksPath() string {
	root_path, err := getRootPath()

	if err != nil {
		panic("Current working directory doesn't exist")
	}
	folder_path := filepath.Join(root_path, "artworks")
	return folder_path
}

func getRootPath() (string, error) {
	cwd_path, err := os.Getwd()
	root_path := filepath.Dir(filepath.Dir(cwd_path))
	return root_path, err
}

func queryLLM(ctx context.Context, g *genkit.Genkit, outputSchema map[string]any, artwork string) (*ai.ModelResponse, error) {
	start := time.Now()
	resp, err := genkit.Generate(ctx, g,
		ai.WithModelName(MODEL_NAME),
		ai.WithSystem(systemDefinition),
		ai.WithPrompt(BuildPrompt(artwork)),
		ai.WithOutputSchema(outputSchema),
		ai.WithConfig(map[string]any{
			"Temperature": 0.0,
		}),
	)
	if err != nil {
		log.Println("error:", err)
		// panic(err)
		// resp := queryLLM(ctx, g, outputSchema, artwork)
	}
	duration := time.Since(start)
	fmt.Printf("Duration: %v", duration)
	return resp, err
}

func setupLLMConfiguration() (context.Context, *genkit.Genkit) {
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

	// firebase.EnableFirebaseTelemetry(nil)

	g := genkit.Init(ctx, genkit.WithPlugins(nebulaProvider))
	return ctx, g
}

func BuildPrompt(artwork string) string {
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
