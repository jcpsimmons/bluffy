package main

import (
	"fmt"
	"log"
	"os"

	"github.com/simsies/blog/cli/pkg/database"
	"github.com/simsies/blog/cli/pkg/embedding"
	"github.com/simsies/blog/cli/pkg/similarity"
	"github.com/simsies/blog/cli/pkg/textproc"
	"github.com/spf13/cobra"
)

func main() {
	var inputFile string
	var outputDir string
	var maxWorkers int

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
			
			if err := processFile(inputFile, outputDir, maxWorkers); err != nil {
				log.Fatalf("Error processing file: %v", err)
			}
		},
	}

	rootCmd.Flags().StringVarP(&inputFile, "file", "f", "", "Input text file (.txt or .md)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for the SQLite database")
	rootCmd.Flags().IntVarP(&maxWorkers, "workers", "w", 0, "Maximum number of concurrent workers (0 = number of CPUs)")
	rootCmd.MarkFlagRequired("file")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func processFile(inputFile, outputDir string, maxWorkers int) error {
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

	client := embedding.NewOllamaClient("", "")

	fmt.Printf("Generating embeddings with %d workers...\n", maxWorkers)
	
	processedChunks, err := client.GetEmbeddingsConcurrent(chunks, maxWorkers)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

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