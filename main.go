package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

type TextChunk struct {
	ID        int     `json:"id"`
	Text      string  `json:"text"`
	ChunkIndex int    `json:"chunk_index"`
	Embedding []float64 `json:"embedding"`
}

type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func main() {
	var inputFile string
	var outputDir string

	rootCmd := &cobra.Command{
		Use:   "embed-cli",
		Short: "Generate embeddings for text chunks using Nomic on Ollama",
		Long:  "A CLI tool that processes text files, chunks them by paragraphs, and generates embeddings using Nomic running on Ollama locally.",
		Run: func(cmd *cobra.Command, args []string) {
			if inputFile == "" {
				fmt.Println("Error: input file is required")
				cmd.Help()
				os.Exit(1)
			}
			
			if outputDir == "" {
				outputDir = "."
			}
			
			if err := processFile(inputFile, outputDir); err != nil {
				log.Fatalf("Error processing file: %v", err)
			}
		},
	}

	rootCmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input text file (.txt or .md)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for the SQLite database")
	rootCmd.MarkFlagRequired("file")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func processFile(inputFile, outputDir string) error {
	chunks, err := chunkTextByParagraphs(inputFile)
	if err != nil {
		return fmt.Errorf("failed to chunk text: %w", err)
	}

	fmt.Printf("Processed %d text chunks\n", len(chunks))

	dbPath, err := createOutputDatabase(inputFile, outputDir)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := setupDatabase(db); err != nil {
		return fmt.Errorf("failed to setup database: %w", err)
	}

	for i, chunk := range chunks {
		fmt.Printf("Processing chunk %d/%d...\n", i+1, len(chunks))
		
		embedding, err := getEmbedding(chunk.Text)
		if err != nil {
			return fmt.Errorf("failed to get embedding for chunk %d: %w", i, err)
		}

		chunk.Embedding = embedding

		if err := insertChunk(db, chunk); err != nil {
			return fmt.Errorf("failed to insert chunk %d: %w", i, err)
		}
	}

	fmt.Printf("Successfully processed all chunks and stored embeddings in database: %s\n", dbPath)
	fmt.Println("Database is ready for exploration with any SQLite browser.")
	
	return nil
}

func chunkTextByParagraphs(filename string) ([]TextChunk, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunks []TextChunk
	var currentChunk strings.Builder
	chunkIndex := 0
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, TextChunk{
					Text:       strings.TrimSpace(currentChunk.String()),
					ChunkIndex: chunkIndex,
				})
				currentChunk.Reset()
				chunkIndex++
			}
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(line)
		}
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, TextChunk{
			Text:       strings.TrimSpace(currentChunk.String()),
			ChunkIndex: chunkIndex,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return chunks, nil
}

func getEmbedding(text string) ([]float64, error) {
	reqBody := OllamaEmbeddingRequest{
		Model:  "nomic-embed-text",
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("http://localhost:11434/api/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embedding, nil
}

func createOutputDatabase(inputFile, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}
	
	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	dbPath := filepath.Join(outputDir, fmt.Sprintf("%s_embeddings.db", baseName))
	return dbPath, nil
}

func setupDatabase(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS text_chunks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		text TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		embedding TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(query)
	return err
}

func insertChunk(db *sql.DB, chunk TextChunk) error {
	embeddingJSON, err := json.Marshal(chunk.Embedding)
	if err != nil {
		return err
	}

	query := `INSERT INTO text_chunks (text, chunk_index, embedding) VALUES (?, ?, ?)`
	_, err = db.Exec(query, chunk.Text, chunk.ChunkIndex, string(embeddingJSON))
	return err
}