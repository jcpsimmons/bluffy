package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"

	"github.com/simsies/blog/cli/pkg/database"
)

type OllamaClient struct {
	baseURL string
	model   string
}

type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type EmbeddingJob struct {
	Index int
	Chunk *database.TextChunk
}

type EmbeddingResult struct {
	Index int
	Chunk *database.TextChunk
	Error error
}

type SummaryJob struct {
	Index int
	Chunk *database.TextChunk
}

type SummaryResult struct {
	Index int
	Chunk *database.TextChunk
	Error error
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}

	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
	}
}

func (c *OllamaClient) GetEmbedding(text string) ([]float64, error) {
	reqBody := embeddingRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", c.baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embedding, nil
}

func (c *OllamaClient) GetSummary(text string) (string, error) {
	prompt := fmt.Sprintf("Summarize this text in exactly 1-5 words (no punctuation, just the key topic):\n\n%s", text)
	
	reqBody := generateRequest{
		Model:  "qwen2.5:0.5b",
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", c.baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Clean up the response - remove extra whitespace and limit to 5 words
	summary := strings.TrimSpace(result.Response)
	words := strings.Fields(summary)
	if len(words) > 5 {
		words = words[:5]
	}
	
	return strings.Join(words, " "), nil
}

func (c *OllamaClient) GetEmbeddingsConcurrent(chunks []database.TextChunk, maxWorkers int) ([]database.TextChunk, error) {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	jobs := make(chan EmbeddingJob, len(chunks))
	results := make(chan EmbeddingResult, len(chunks))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go c.worker(jobs, results, &wg)
	}

	// Send jobs
	for i, chunk := range chunks {
		jobs <- EmbeddingJob{Index: i, Chunk: &chunk}
	}
	close(jobs)

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	processedChunks := make([]database.TextChunk, len(chunks))
	var errors []error

	for result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("chunk %d: %w", result.Index, result.Error))
		} else {
			processedChunks[result.Index] = *result.Chunk
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("embedding errors occurred: %v", errors)
	}

	return processedChunks, nil
}

func (c *OllamaClient) GetSummariesConcurrent(chunks []database.TextChunk, maxWorkers int) ([]database.TextChunk, error) {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	jobs := make(chan SummaryJob, len(chunks))
	results := make(chan SummaryResult, len(chunks))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go c.summaryWorker(jobs, results, &wg)
	}

	// Send jobs
	for i, chunk := range chunks {
		jobs <- SummaryJob{Index: i, Chunk: &chunk}
	}
	close(jobs)

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	processedChunks := make([]database.TextChunk, len(chunks))
	var errors []error

	for result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("chunk %d: %w", result.Index, result.Error))
		} else {
			processedChunks[result.Index] = *result.Chunk
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("summarization errors occurred: %v", errors)
	}

	return processedChunks, nil
}

func (c *OllamaClient) worker(jobs <-chan EmbeddingJob, results chan<- EmbeddingResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		embedding, err := c.GetEmbedding(job.Chunk.Text)
		if err != nil {
			results <- EmbeddingResult{Index: job.Index, Error: err}
			continue
		}

		job.Chunk.Embedding = embedding
		results <- EmbeddingResult{Index: job.Index, Chunk: job.Chunk}
	}
}

func (c *OllamaClient) summaryWorker(jobs <-chan SummaryJob, results chan<- SummaryResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		summary, err := c.GetSummary(job.Chunk.Text)
		if err != nil {
			results <- SummaryResult{Index: job.Index, Error: err}
			continue
		}

		job.Chunk.Summary = summary
		results <- SummaryResult{Index: job.Index, Chunk: job.Chunk}
	}
}