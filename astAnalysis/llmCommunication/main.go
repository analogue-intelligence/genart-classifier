package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

// var MODEL_NAME = "nebula/deepseek-r1:8b"

// var MODEL_NAME = "nebula/deepseek-r1:1.5b"
var MODEL_NAME = "nebula/FAST.gemma3:12b"

type ArtworkClassification struct {
	P5Keywords        []string `json:"p5_keywords_identified"`
	MaterialProcesses []string `json:"material_and_processes"`
	Interaction       []string `json:"interaction"`
	Outcome           []string `json:"outcomes"`
	Explanation       string   `json:"logic_explanation"`
	Algorithm         string   `json:"reuse_algorithm"`
}

type EnrichedArtworkClassification struct {
	ArtworkClassification
	InputTokens  int   `json:"input_tokens"`
	OutputTokens int   `json:"output_tokens"`
	LatencyMs    int64 `json:"latency_ms"`
}

func main() {
	fmt.Println("Runing Script...")
	ctx, g := setupLLMConfiguration()

	// outputSchema := map[string]any{
	// 	"material_and_processes": "",
	// 	"interaction":            "",
	// 	"outcome":                "",
	// 	"explanation":            "",
	// 	"reuse_algorithm":        "",
	// }

	artworks_path := getArtworksPath()

	readFolder(artworks_path, ctx, g)
}

func readFolder(artworks_path string, ctx context.Context, g *genkit.Genkit) {
	err := filepath.WalkDir(artworks_path, func(filep string, info os.DirEntry, err error) error {
		if !info.IsDir() && filepath.Ext(filep) == ".js" {
			fmt.Println("Reading artwork: " + info.Name())

			artworkSourceCode, err := os.ReadFile(filep)

			if err != nil {
				panic("Reading the content of the file doesn't work.")
			}
			response, _ := queryLLM(ctx, g, string(artworkSourceCode), info.Name())

			processResponse(response, info.Name())
			// return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		panic("Error while walking the directory")
	}
}

func processResponse(response *ai.ModelResponse, filename string) {
	response_path := createFile(filename)

	var classification ArtworkClassification
	response.Output(&classification)

	usage := response.Usage // input/output token counts

	enriched := EnrichedArtworkClassification{
		ArtworkClassification: classification,
		InputTokens:           usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		LatencyMs:             int64(response.LatencyMs),
	}
	data, err := json.Marshal(enriched)
	if err != nil {
		log.Println(err)
	}

	// if err := os.WriteFile(response_path, []byte(response.Text()), 0644); err != nil {
	if err := os.WriteFile(response_path, []byte(data), 0644); err != nil {
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

	// Owner: rwx (7) = 4 (read) + 2 (write) + 1 (execute)
	// Group: r-x (5) =  4 (read) + 0 (no write) + 1 (execute)
	// Others: r-x (5) =  4 (read) + 0 (no write) + 1 (execute)
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		panic(err)
	}

	tmp_path := filepath.Join(outputPath, filename)

	response_path := strings.TrimSuffix(tmp_path, filepath.Ext(tmp_path)) + ".json"

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

func queryWithRetry(ctx context.Context, g *genkit.Genkit, maxRetries int, opts ...ai.GenerateOption) (*ai.ModelResponse, error) {
	var lastErr error
	for i := range maxRetries {
		resp, err := genkit.Generate(ctx, g, opts...)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		log.Println("generation attempt failed, retrying", "attempt", i+1, "err", err)
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	return nil, fmt.Errorf("all %d attempts failed: %w", maxRetries, lastErr)
}

func queryLLM(ctx context.Context, g *genkit.Genkit, artwork string, filename string) (*ai.ModelResponse, error) {
	resp, err := queryWithRetry(ctx, g, 4,
		ai.WithModelName(MODEL_NAME),
		ai.WithSystem(systemDefinition),
		ai.WithPrompt(BuildPrompt(artwork, filename)),
		ai.WithOutputType(ArtworkClassification{}),
		ai.WithConfig(map[string]any{"Temperature": 0.0}),
	)

	// resp, err := genkit.Generate(ctx, g,
	// 	ai.WithModelName(MODEL_NAME),
	// 	ai.WithSystem(systemDefinition),
	// 	// ai.WithPrompt(BuildPrompt(artwork)),
	// 	ai.WithPrompt(BuildPrompt(exampleProgram, filename)),
	// 	// ai.WithOutputSchema(outputSchema),
	// 	ai.WithOutputType(ArtworkClassification{}),
	// 	ai.WithConfig(map[string]any{
	// 		"Temperature": 0.0,
	// 	}),
	// )

	if err != nil {
		log.Println("error:", err)
	}
	fmt.Println(resp.Text())

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

func BuildPrompt(artwork string, filename string) string {
	tmpl, _ := template.New("prompt").Parse(baselineTemplate)

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Artwork":   artwork,
		"Extension": ".js",
		"File_name": filename,
	}
	tmpl.Execute(&buf, data)

	// fmt.Printf("PROMPT: \n %s", buf.String())
	return buf.String()
}
