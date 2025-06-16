package textproc

import (
	"io"
	"os"
	"strings"

	"github.com/jcpsimmons/bluffy/pkg/database"
	"github.com/tmc/langchaingo/textsplitter"
)

func ChunkTextByParagraphs(filename string) ([]database.TextChunk, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read entire file
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	text := string(content)
	return chunkTextWithSplitter(text)
}

func chunkTextWithSplitter(text string) ([]database.TextChunk, error) {
	// Clean up the text
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return nil, nil
	}

	// Create a recursive character text splitter
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(7500),        // A bit under 8192 for safety
		textsplitter.WithChunkOverlap(750),      // 10% overlap (750 chars)
		textsplitter.WithSeparators([]string{    // Custom separators for better text splitting
			"\n\n",    // Paragraph breaks
			"\n",      // Line breaks
			". ",      // Sentence endings
			"! ",
			"? ",
			"; ",      // Clause separators
			", ",      // Comma separators
			" ",       // Word boundaries
			"",        // Character level (fallback)
		}),
	)

	// Split the text into chunks
	docs, err := splitter.SplitText(text)
	if err != nil {
		return nil, err
	}

	// Convert to our TextChunk format
	var chunks []database.TextChunk
	for i, doc := range docs {
		chunk := strings.TrimSpace(doc)
		if len(chunk) > 0 {
			chunks = append(chunks, database.TextChunk{
				Text:       chunk,
				ChunkIndex: i,
			})
		}
	}

	return chunks, nil
}