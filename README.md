# Knowledge Graph CLI

This directory contains a terminal-based CLI implementation for the knowledge graph functionality. The knowledge graph is a visualization tool that helps understand relationships between different pieces of content in the codebase.

## Features

- Terminal-based interactive graph visualization
- Node and edge filtering based on similarity thresholds
- Radial force adjustment for graph layout control
- Node selection and detailed information display
- Keyboard-based navigation and controls

## Implementation

The CLI is built using:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for terminal styling
- ASCII art for graph visualization

## Data Structure

The graph uses the following data structure:

```go
type Node struct {
    ID        string  `json:"id"`
    File      string  `json:"file"`
    Title     string  `json:"title"`
    Paragraph string  `json:"paragraph"`
    FullText  string  `json:"fullText"`
    X         float64 `json:"x,omitempty"`
    Y         float64 `json:"y,omitempty"`
    VX        float64 `json:"vx,omitempty"`
    VY        float64 `json:"vy,omitempty"`
}

type Edge struct {
    Source     string  `json:"source"`
    Target     string  `json:"target"`
    Similarity float64 `json:"similarity"`
}

type GraphData struct {
    Nodes    []Node `json:"nodes"`
    Edges    []Edge `json:"edges"`
    Metadata struct {
        TotalParagraphs  int     `json:"totalParagraphs"`
        TotalConnections int     `json:"totalConnections"`
        Threshold       float64 `json:"threshold"`
        GeneratedAt     string  `json:"generatedAt"`
    } `json:"metadata"`
}
```

## Usage

1. Ensure you have Go 1.21 or later installed:
   ```bash
   go version
   ```

2. Install the CLI:
   ```bash
   go install github.com/simsies/blog/cli/cmd/knowledge-graph@latest
   ```

3. Generate the embeddings data if you haven't already:
   ```bash
   node scripts/generate-embeddings.js
   ```

4. Run the CLI:
   ```bash
   knowledge-graph
   ```

## Controls

- **Arrow Keys**:
  - Up/Down: Adjust radial force
  - Left/Right: Adjust similarity threshold
- **Q**: Quit the application
- **Mouse**: Click nodes to select them and view details

## Development

To modify the graph behavior, you can adjust:
- Force simulation parameters in the `renderGraph` function
- Node and edge styling in the ASCII art rendering
- Layout parameters like initial zoom and center position

## Building from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/simsies/blog.git
   cd blog/cli
   ```

2. Build the CLI:
   ```bash
   go build -o knowledge-graph ./cmd/knowledge-graph
   ```

3. Run the CLI:
   ```bash
   ./knowledge-graph
   ```
