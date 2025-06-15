package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
	path string
}

func NewDB(inputFile, outputDir string) (*DB, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	dbPath := filepath.Join(outputDir, fmt.Sprintf("%s_embeddings.db", baseName))

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	if err := db.setupTables(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to setup database tables: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Path() string {
	return db.path
}

func (db *DB) setupTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS text_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			embedding TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS chunk_similarities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chunk_id_1 INTEGER NOT NULL,
			chunk_id_2 INTEGER NOT NULL,
			distance REAL NOT NULL,
			similarity REAL NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chunk_id_1) REFERENCES text_chunks (id),
			FOREIGN KEY (chunk_id_2) REFERENCES text_chunks (id),
			UNIQUE(chunk_id_1, chunk_id_2)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_similarities_chunk1 ON chunk_similarities(chunk_id_1)`,
		`CREATE INDEX IF NOT EXISTS idx_similarities_chunk2 ON chunk_similarities(chunk_id_2)`,
		`CREATE INDEX IF NOT EXISTS idx_similarities_distance ON chunk_similarities(distance)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	return nil
}

func (db *DB) InsertChunk(chunk *TextChunk) error {
	embeddingJSON, err := json.Marshal(chunk.Embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	query := `INSERT INTO text_chunks (text, chunk_index, embedding) VALUES (?, ?, ?) RETURNING id`
	err = db.conn.QueryRow(query, chunk.Text, chunk.ChunkIndex, string(embeddingJSON)).Scan(&chunk.ID)
	if err != nil {
		return fmt.Errorf("failed to insert chunk: %w", err)
	}

	return nil
}

func (db *DB) GetAllChunks() ([]TextChunk, error) {
	query := `SELECT id, text, chunk_index, embedding FROM text_chunks ORDER BY chunk_index`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []TextChunk
	for rows.Next() {
		var chunk TextChunk
		var embeddingJSON string

		if err := rows.Scan(&chunk.ID, &chunk.Text, &chunk.ChunkIndex, &embeddingJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if err := json.Unmarshal([]byte(embeddingJSON), &chunk.Embedding); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedding for chunk %d: %w", chunk.ID, err)
		}

		chunks = append(chunks, chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return chunks, nil
}

func (db *DB) InsertSimilarity(similarity *ChunkSimilarity) error {
	query := `INSERT INTO chunk_similarities (chunk_id_1, chunk_id_2, distance, similarity) VALUES (?, ?, ?, ?)`
	_, err := db.conn.Exec(query, similarity.ChunkID1, similarity.ChunkID2, similarity.Distance, similarity.Similarity)
	if err != nil {
		return fmt.Errorf("failed to insert similarity: %w", err)
	}
	return nil
}

func (db *DB) BatchInsertSimilarities(similarities []ChunkSimilarity) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO chunk_similarities (chunk_id_1, chunk_id_2, distance, similarity) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, similarity := range similarities {
		if _, err := stmt.Exec(similarity.ChunkID1, similarity.ChunkID2, similarity.Distance, similarity.Similarity); err != nil {
			return fmt.Errorf("failed to insert similarity %d-%d: %w", similarity.ChunkID1, similarity.ChunkID2, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}