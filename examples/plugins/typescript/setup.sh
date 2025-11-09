#!/bin/bash
set -e

echo "🚀 Setting up TypeScript Dragonfly Plugin..."

# Check for Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js first."
    exit 1
fi

echo "✅ Node.js found: $(node --version)"

# Check for protoc
if ! command -v protoc &> /dev/null; then
    echo "⚠️  protoc not found. Installing via npm..."
    npm install -g protoc-gen-ts_proto
    
    # Check if Homebrew is available (macOS)
    if command -v brew &> /dev/null; then
        echo "Installing protoc via Homebrew..."
        brew install protobuf
    else
        echo ""
        echo "❌ Please install protoc manually:"
        echo "   macOS: brew install protobuf"
        echo "   Linux: sudo apt install protobuf-compiler"
        echo "   Or download from: https://github.com/protocolbuffers/protobuf/releases"
        exit 1
    fi
fi

echo "✅ protoc found: $(protoc --version)"

# Install npm dependencies
echo ""
echo "📦 Installing npm dependencies..."
npm install

# Generate TypeScript types from proto
echo ""
echo "🔨 Generating TypeScript types from protobuf schema..."
npm run generate

if [ $? -ne 0 ]; then
    echo ""
    echo "⚠️  Type generation failed. This is optional - you can still use the plugin."
    echo "    The plugin will work with runtime proto loading instead."
fi

# Build TypeScript
echo ""
echo "🏗️  Building TypeScript..."
npm run build

echo ""
echo "✅ Setup complete!"
echo ""
echo "To run the plugin:"
echo "  npm run dev    # Development mode with hot reload"
echo "  npm start      # Production mode"
echo ""
echo "To enable in Dragonfly, edit plugins/plugins.yaml and uncomment the example-typescript section"

