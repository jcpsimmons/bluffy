package database

type TextChunk struct {
	ID         int       `json:"id"`
	Text       string    `json:"text"`
	ChunkIndex int       `json:"chunk_index"`
	Embedding  []float64 `json:"embedding"`
}

type ChunkSimilarity struct {
	ID           int     `json:"id"`
	ChunkID1     int     `json:"chunk_id_1"`
	ChunkID2     int     `json:"chunk_id_2"`
	Distance     float64 `json:"distance"`
	Similarity   float64 `json:"similarity"`
}