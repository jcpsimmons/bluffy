package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/simsies/blog/cli/pkg/database"
	"github.com/simsies/blog/cli/pkg/embedding"
	"github.com/simsies/blog/cli/pkg/similarity"
	"github.com/simsies/blog/cli/pkg/textproc"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "embed-cli",
		Short: "Generate embeddings for text chunks using Nomic on Ollama",
		Long:  "A CLI tool that processes text files, chunks them by paragraphs, and generates embeddings using Nomic running on Ollama locally.",
	}

	// Add subcommands
	rootCmd.AddCommand(createProcessCommand())
	rootCmd.AddCommand(createServeCommand())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func createProcessCommand() *cobra.Command {
	var inputFile string
	var outputDir string
	var maxWorkers int
	var ollamaHost string

	cmd := &cobra.Command{
		Use:   "process",
		Short: "Process text file and generate embeddings",
		Long:  "Process a text file, chunk it by paragraphs, generate embeddings and summaries, and store in SQLite database.",
		Run: func(cmd *cobra.Command, args []string) {
			if inputFile == "" {
				fmt.Println("Error: input file is required")
				cmd.Help()
				os.Exit(1)
			}

			if outputDir == "" {
				outputDir = "."
			}

			if err := processFile(inputFile, outputDir, maxWorkers, ollamaHost); err != nil {
				log.Fatalf("Error processing file: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input text file (.txt or .md)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for the SQLite database")
	cmd.Flags().IntVarP(&maxWorkers, "workers", "w", 0, "Maximum number of concurrent workers (0 = number of CPUs)")
	cmd.Flags().StringVar(&ollamaHost, "ollama-host", "http://localhost:11434", "Ollama server host and port")
	cmd.MarkFlagRequired("file")

	return cmd
}

func createServeCommand() *cobra.Command {
	var dbPath string
	var port int

	cmd := &cobra.Command{
		Use:   "serve <database.db>",
		Short: "Start API server for embeddings database",
		Long:  "Start a REST API server to serve the embeddings database for visualization and analysis.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dbPath = args[0]
			if err := startAPIServer(dbPath, port); err != nil {
				log.Fatalf("Error starting API server: %v", err)
			}
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")

	return cmd
}

func processFile(inputFile, outputDir string, maxWorkers int, ollamaHost string) error {
	chunks, err := textproc.ChunkTextByParagraphs(inputFile)
	if err != nil {
		return fmt.Errorf("failed to chunk text: %w", err)
	}

	fmt.Printf("Processed %d text chunks\n", len(chunks))

	db, err := database.NewDB(inputFile, outputDir)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	client := embedding.NewOllamaClient(ollamaHost, "")

	// Check Ollama connectivity and model availability
	fmt.Printf("Checking Ollama connectivity...\n")
	if err := client.CheckConnection(); err != nil {
		return err
	}

	fmt.Printf("Checking required models...\n")
	if err := client.CheckModelsAvailable(); err != nil {
		return err
	}

	// Set default workers if not specified
	if maxWorkers <= 0 {
		maxWorkers = 1
	}

	fmt.Printf("Generating embeddings with %d workers...\n", maxWorkers)

	processedChunks, err := client.GetEmbeddingsConcurrent(chunks, maxWorkers, func(completed, total int) {
		printProgressBar("Embeddings", completed, total)
	})
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	fmt.Println() // New line after progress bar

	fmt.Printf("Generating summaries with %d workers...\n", maxWorkers)

	processedChunks, err = client.GetSummariesConcurrent(processedChunks, maxWorkers, func(completed, total int) {
		printProgressBar("Summaries", completed, total)
	})
	if err != nil {
		return fmt.Errorf("failed to generate summaries: %w", err)
	}
	fmt.Println() // New line after progress bar

	fmt.Println("Storing chunks in database...")

	for i, chunk := range processedChunks {
		if err := db.InsertChunk(&chunk); err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}
		processedChunks[i] = chunk
	}

	fmt.Println("Calculating similarities between all chunks...")

	similarities, err := similarity.CalculateAllSimilarities(processedChunks)
	if err != nil {
		return fmt.Errorf("failed to calculate similarities: %w", err)
	}

	fmt.Printf("Storing %d similarity calculations...\n", len(similarities))

	if err := db.BatchInsertSimilarities(similarities); err != nil {
		return fmt.Errorf("failed to store similarities: %w", err)
	}

	fmt.Printf("Successfully processed all chunks and stored embeddings in database: %s\n", db.Path())
	fmt.Printf("Calculated and stored %d chunk similarities\n", len(similarities))
	fmt.Println("Database is ready for exploration with any SQLite browser.")

	return nil
}

func printProgressBar(prefix string, completed, total int) {
	width := 50
	percentage := float64(completed) / float64(total)
	filled := int(percentage * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	fmt.Printf("\r%s: [%s] %d/%d (%.1f%%)",
		prefix, bar, completed, total, percentage*100)
}

// API Server Types and Functions
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type GraphData struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type Node struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Index   int    `json:"index"`
	Summary string `json:"summary"`
}

type Link struct {
	Source     int     `json:"source"`
	Target     int     `json:"target"`
	Distance   float64 `json:"distance"`
	Similarity float64 `json:"similarity"`
}

type APIServer struct {
	dbPath string
}

func startAPIServer(dbPath string, port int) error {
	server := &APIServer{dbPath: dbPath}

	http.HandleFunc("/api/chunks", enableCORS(server.handleChunks))
	http.HandleFunc("/api/similarities", enableCORS(server.handleSimilarities))
	http.HandleFunc("/api/graph", enableCORS(server.handleGraph))

	log.Printf("Starting API server on port %d", port)
	log.Printf("Database: %s", dbPath)
	log.Printf("Endpoints:")
	log.Printf("  GET /api/chunks - Get all text chunks")
	log.Printf("  GET /api/similarities - Get all similarities")
	log.Printf("  GET /api/graph - Get graph data for visualization")

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *APIServer) openDB() (*database.DB, error) {
	return database.OpenExistingDB(s.dbPath)
}

func (s *APIServer) handleChunks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := s.openDB()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	chunks, err := db.GetAllChunks()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get chunks: %v", err), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, chunks)
}

func (s *APIServer) handleSimilarities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := s.openDB()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	similarities, err := db.GetAllSimilarities()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get similarities: %v", err), http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, similarities)
}

func (s *APIServer) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	minSimilarity := 0.0
	if sim := r.URL.Query().Get("min_similarity"); sim != "" {
		if parsed, err := strconv.ParseFloat(sim, 64); err == nil {
			minSimilarity = parsed
		}
	}

	db, err := s.openDB()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	chunks, err := db.GetAllChunks()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get chunks: %v", err), http.StatusInternalServerError)
		return
	}

	similarities, err := db.GetAllSimilarities()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get similarities: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to graph format
	nodes := make([]Node, len(chunks))
	for i, chunk := range chunks {
		nodes[i] = Node{
			ID:      chunk.ID,
			Text:    chunk.Text,
			Index:   chunk.ChunkIndex,
			Summary: chunk.Summary,
		}
	}

	var links []Link
	for _, sim := range similarities {
		if sim.Similarity >= minSimilarity {
			links = append(links, Link{
				Source:     sim.ChunkID1,
				Target:     sim.ChunkID2,
				Distance:   sim.Distance,
				Similarity: sim.Similarity,
			})
		}
	}

	graphData := GraphData{
		Nodes: nodes,
		Links: links,
	}

	respondWithJSON(w, graphData)
}

func enableCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

func respondWithJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	response := APIResponse{
		Success: true,
		Data:    data,
	}
	json.NewEncoder(w).Encode(response)
}

func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := APIResponse{
		Success: false,
		Error:   message,
	}
	json.NewEncoder(w).Encode(response)
}
