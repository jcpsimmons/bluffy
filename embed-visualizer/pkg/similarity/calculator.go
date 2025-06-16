package similarity

import (
	"fmt"
	"math"

	"embed-visualizer/pkg/database"
)

func CosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length: %d vs %d", len(a), len(b))
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return dotProduct / (normA * normB), nil
}

func EuclideanDistance(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length: %d vs %d", len(a), len(b))
	}

	var sum float64
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum), nil
}

func CalculateAllSimilarities(chunks []database.TextChunk) ([]database.ChunkSimilarity, error) {
	var similarities []database.ChunkSimilarity

	for i := 0; i < len(chunks); i++ {
		for j := i + 1; j < len(chunks); j++ {
			chunk1 := chunks[i]
			chunk2 := chunks[j]

			distance, err := EuclideanDistance(chunk1.Embedding, chunk2.Embedding)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate distance between chunks %d and %d: %w", chunk1.ID, chunk2.ID, err)
			}

			cosineSim, err := CosineSimilarity(chunk1.Embedding, chunk2.Embedding)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate similarity between chunks %d and %d: %w", chunk1.ID, chunk2.ID, err)
			}

			similarity := database.ChunkSimilarity{
				ChunkID1:   chunk1.ID,
				ChunkID2:   chunk2.ID,
				Distance:   distance,
				Similarity: cosineSim,
			}

			similarities = append(similarities, similarity)
		}
	}

	return similarities, nil
}