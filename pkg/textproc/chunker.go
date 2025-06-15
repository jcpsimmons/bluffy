package textproc

import (
	"bufio"
	"os"
	"strings"

	"github.com/simsies/blog/cli/pkg/database"
)

func ChunkTextByParagraphs(filename string) ([]database.TextChunk, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunks []database.TextChunk
	var currentChunk strings.Builder
	chunkIndex := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, database.TextChunk{
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
		chunks = append(chunks, database.TextChunk{
			Text:       strings.TrimSpace(currentChunk.String()),
			ChunkIndex: chunkIndex,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return chunks, nil
}