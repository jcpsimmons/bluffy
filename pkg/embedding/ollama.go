package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
	prompt := fmt.Sprintf("Please provide only a 1-5 word summary of this text. Do not include any reasoning, explanations, or thinking process. Limit your response to a maximum of 5 words. Just respond with the key topic:\n\n%s \n\n /no_think", text)

	reqBody := generateRequest{
		Model:  "qwen3:0.6b",
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

	// Clean up the response - remove thinking tags and clean text
	summary := cleanSummaryResponse(result.Response)
	words := strings.Fields(summary)
	if len(words) > 10 {
		words = words[:10]
	}

	return strings.Join(words, " "), nil
}

func cleanSummaryResponse(response string) string {
	// Remove <think> tags and their content
	thinkRegex := regexp.MustCompile(`(?s)<think>.*?</think>`)
	cleaned := thinkRegex.ReplaceAllString(response, "")

	// Remove any remaining XML-like tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	cleaned = tagRegex.ReplaceAllString(cleaned, "")

	// Clean up whitespace and common prefixes
	cleaned = strings.TrimSpace(cleaned)

	// Remove common response prefixes
	prefixes := []string{
		"Summary:", "Topic:", "Key words:", "Keywords:",
		"The text is about", "This text discusses", "The topic is",
		"Main topic:", "Subject:", "Theme:",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(cleaned), strings.ToLower(prefix)) {
			cleaned = strings.TrimSpace(cleaned[len(prefix):])
			break
		}
	}

	// Remove punctuation from the end
	punctuation := []string{".", "!", "?", ":", ";", ","}
	for _, punct := range punctuation {
		cleaned = strings.TrimSuffix(cleaned, punct)
	}

	return strings.TrimSpace(cleaned)
}

func (c *OllamaClient) GetEmbeddingsConcurrent(chunks []database.TextChunk, maxWorkers int, progressCallback func(completed, total int)) ([]database.TextChunk, error) {
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

	// Collect results with progress tracking
	processedChunks := make([]database.TextChunk, len(chunks))
	var errors []error
	completed := 0
	total := len(chunks)

	for result := range results {
		completed++
		if progressCallback != nil {
			progressCallback(completed, total)
		}

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

func (c *OllamaClient) GetSummariesConcurrent(chunks []database.TextChunk, maxWorkers int, progressCallback func(completed, total int)) ([]database.TextChunk, error) {
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

	// Collect results with progress tracking
	processedChunks := make([]database.TextChunk, len(chunks))
	var errors []error
	completed := 0
	total := len(chunks)

	for result := range results {
		completed++
		if progressCallback != nil {
			progressCallback(completed, total)
		}

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
