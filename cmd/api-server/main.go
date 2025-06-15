package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/simsies/blog/cli/pkg/database"
)

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

func main() {
	var dbPath string
	var port int

	flag.StringVar(&dbPath, "db", "", "Path to SQLite database file")
	flag.IntVar(&port, "port", 8080, "Server port")
	flag.Parse()

	if dbPath == "" {
		log.Fatal("Database path is required. Use -db flag.")
	}

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
	
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

type APIServer struct {
	dbPath string
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