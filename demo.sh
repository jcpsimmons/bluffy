#!/bin/bash

# Exit on any error
set -e

# Check for required dependencies
echo "🔍 Checking dependencies..."

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later from https://golang.org/dl/"
    exit 1
fi

# Check for npm
if ! command -v npm &> /dev/null; then
    echo "❌ npm is not installed. Please install Node.js and npm from https://nodejs.org/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"
if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is too old. Please install Go 1.21 or later"
    exit 1
fi

echo "✅ All dependencies are installed"

echo "🚀 Starting demo..."

# Process the Karamazov text
echo "📚 Processing Brothers Karamazov..."
./bluffy process -f examples/corpus/karamazov.txt -w 2

# Get the database file name
DB_FILE="karamazov_embeddings.db"

# Start the server in the background
echo "🌐 Starting server..."
./bluffy serve "$DB_FILE" &
SERVER_PID=$!

# Wait a moment for the server to start
sleep 2

# Start the visualizer
echo "🎨 Starting visualizer..."
cd examples/visualizer
npm install
npm start &
VISUALIZER_PID=$!
cd ../..

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up..."
    kill $SERVER_PID 2>/dev/null || true
    kill $VISUALIZER_PID 2>/dev/null || true
}

# Set up cleanup on script exit
trap cleanup EXIT

# Keep the script running until user interrupts
echo "✨ Demo is running! Press Ctrl+C to stop."
wait 
