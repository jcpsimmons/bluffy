package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func OpenExistingDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	return db, nil
}

func (db *DB) GetAllSimilarities() ([]ChunkSimilarity, error) {
	query := `SELECT id, chunk_id_1, chunk_id_2, distance, similarity FROM chunk_similarities ORDER BY similarity DESC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query similarities: %w", err)
	}
	defer rows.Close()

	var similarities []ChunkSimilarity
	for rows.Next() {
		var sim ChunkSimilarity
		if err := rows.Scan(&sim.ID, &sim.ChunkID1, &sim.ChunkID2, &sim.Distance, &sim.Similarity); err != nil {
			return nil, fmt.Errorf("failed to scan similarity row: %w", err)
		}
		similarities = append(similarities, sim)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating similarity rows: %w", err)
	}

	return similarities, nil
}