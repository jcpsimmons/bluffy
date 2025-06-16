package main

import (
	"context"
	"fmt"
	"os"

	"embed-visualizer/pkg/database"
	"embed-visualizer/pkg/embedding"
	"embed-visualizer/pkg/textproc"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
	db  *database.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ProcessFile processes a text file and generates embeddings
func (a *App) ProcessFile(filePath, outputDir, ollamaHost string, maxWorkers int) error {

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create database
	db, err := database.NewDB(filePath, outputDir)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()
	a.db = db

	// Chunk text  
	textChunks, err := textproc.ChunkTextByParagraphs(filePath)
	if err != nil {
		return fmt.Errorf("failed to chunk text: %w", err)
	}

	// Generate embeddings
	client := embedding.NewOllamaClient(ollamaHost, "nomic-embed-text")
	
	// Progress callback for embeddings - capture context
	ctx := a.ctx
	embeddingProgress := func(completed, total int) {
		go func() {
			runtime.EventsEmit(ctx, "embedding-progress", map[string]interface{}{
				"completed": completed,
				"total":     total,
				"message":   fmt.Sprintf("Generating embeddings: %d/%d", completed, total),
			})
		}()
	}

	processedChunks, err := client.GetEmbeddingsConcurrent(textChunks, maxWorkers, embeddingProgress)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Generate summaries
	summaryProgress := func(completed, total int) {
		go func() {
			runtime.EventsEmit(ctx, "summary-progress", map[string]interface{}{
				"completed": completed,
				"total":     total,
				"message":   fmt.Sprintf("Generating summaries: %d/%d", completed, total),
			})
		}()
	}

	processedChunks, err = client.GetSummariesConcurrent(processedChunks, maxWorkers, summaryProgress)
	if err != nil {
		return fmt.Errorf("failed to generate summaries: %w", err)
	}

	// Store in database
	for _, chunk := range processedChunks {
		if err := db.InsertChunk(&chunk); err != nil {
			return fmt.Errorf("failed to store chunk: %w", err)
		}
	}

	// Calculate similarities
	go func() {
		runtime.EventsEmit(ctx, "similarity-progress", map[string]interface{}{
			"message": "Calculating similarities...",
		})
	}()

	if err := db.CalculateSimilarities(); err != nil {
		return fmt.Errorf("failed to calculate similarities: %w", err)
	}

	runtime.EventsEmit(a.ctx, "processing-complete", map[string]interface{}{
		"message": "Processing completed successfully!",
		"dbPath":  db.Path(),
	})

	return nil
}

// OpenDatabase opens an existing database file
func (a *App) OpenDatabase(dbPath string) error {
	db, err := database.OpenExistingDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	
	if a.db != nil {
		a.db.Close()
	}
	a.db = db
	
	return nil
}

// GetGraphData returns graph data for visualization
func (a *App) GetGraphData(minSimilarity float64) (map[string]interface{}, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no database open")
	}

	chunks, err := a.db.GetChunks()
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}

	similarities, err := a.db.GetSimilarities(minSimilarity)
	if err != nil {
		return nil, fmt.Errorf("failed to get similarities: %w", err)
	}

	// Build nodes
	nodes := make([]map[string]interface{}, len(chunks))
	for i, chunk := range chunks {
		nodes[i] = map[string]interface{}{
			"id":      chunk.ChunkIndex,
			"index":   chunk.ChunkIndex,
			"text":    chunk.Text,
			"summary": chunk.Summary,
		}
	}

	// Build links
	var links []map[string]interface{}
	for _, sim := range similarities {
		links = append(links, map[string]interface{}{
			"source":     sim.ChunkID1,
			"target":     sim.ChunkID2,
			"similarity": sim.Similarity,
		})
	}

	return map[string]interface{}{
		"nodes": nodes,
		"links": links,
	}, nil
}

// SelectFile opens a file picker dialog
func (a *App) SelectFile() (string, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Text File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Text Files",
				Pattern:     "*.txt;*.md",
			},
		},
	})
	return filePath, err
}

// SelectDirectory opens a directory picker dialog
func (a *App) SelectDirectory() (string, error) {
	dirPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Output Directory",
	})
	return dirPath, err
}

// SelectDatabase opens a file picker for database files
func (a *App) SelectDatabase() (string, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Database File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Database Files",
				Pattern:     "*.db",
			},
		},
	})
	return filePath, err
}
