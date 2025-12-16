#!/bin/bash

# SDK Generator Script
# This script helps generate SDKs for testing purposes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
SCHEMA="${SCHEMA:-examples/petstore.yaml}"
LANGUAGE="${LANGUAGE:-all}"  # Default to 'all' to generate SDKs for all languages
SDK_NAME="${SDK_NAME:-petstore}"
OUTPUT_DIR="${OUTPUT_DIR:-test-output}"
HTTP_LIB="${HTTP_LIB:-}"

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/bin/sdk-forge"

# Function to print usage
usage() {
    echo -e "${BLUE}Usage:${NC} $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -s, --schema PATH      OpenAPI schema file (default: examples/petstore.yaml)"
    echo "  -l, --lang LANGUAGE    Target language: go, python, php, js, all (default: all)"
    echo "                         Use 'all' to generate SDKs for all available languages"
    echo "  -n, --name NAME        SDK name (default: petstore)"
    echo "  -o, --output DIR       Output directory (default: test-output)"
    echo "  --http-lib LIB         HTTP library (optional)"
    echo "  -f, --force            Force overwrite existing directory"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  SCHEMA, LANGUAGE, SDK_NAME, OUTPUT_DIR, HTTP_LIB"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Generate SDKs for all languages (default)"
    echo "  $0 -l python -n my-api                # Generate Python SDK only"
    echo "  $0 -l go -n my-api                    # Generate Go SDK only"
    echo "  $0 -l all -n my-api                   # Generate all languages (explicit)"
    echo "  $0 -s my-api.yaml -l go -n my-sdk    # Custom schema and name"
    echo ""
}

# Parse command line arguments
FORCE=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--schema)
            SCHEMA="$2"
            shift 2
            ;;
        -l|--lang|--language)
            LANGUAGE="$2"
            shift 2
            ;;
        -n|--name)
            SDK_NAME="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --http-lib)
            HTTP_LIB="$2"
            shift 2
            ;;
        -f|--force)
            FORCE="--force"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Error:${NC} Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check if binary exists
if [[ ! -f "$BINARY" ]]; then
    echo -e "${YELLOW}Binary not found. Building...${NC}"
    cd "$SCRIPT_DIR"
    make build
    echo ""
fi

# Check if schema file exists
if [[ ! -f "$SCHEMA" ]]; then
    echo -e "${RED}Error:${NC} Schema file not found: $SCHEMA"
    exit 1
fi

# Build command
CMD="$BINARY generate --schema \"$SCHEMA\" --lang \"$LANGUAGE\" --name \"$SDK_NAME\" --output \"$OUTPUT_DIR\""

if [[ -n "$HTTP_LIB" ]]; then
    CMD="$CMD --http-lib \"$HTTP_LIB\""
fi

if [[ -n "$FORCE" ]]; then
    CMD="$CMD $FORCE"
fi

# Print configuration
echo -e "${BLUE}=== SDK Generation Configuration ===${NC}"
echo -e "Schema:     ${GREEN}$SCHEMA${NC}"
echo -e "Language:   ${GREEN}$LANGUAGE${NC}"
echo -e "SDK Name:   ${GREEN}$SDK_NAME${NC}"
echo -e "Output:     ${GREEN}$OUTPUT_DIR${NC}"
if [[ -n "$HTTP_LIB" ]]; then
    echo -e "HTTP Lib:   ${GREEN}$HTTP_LIB${NC}"
fi
echo ""

# Execute command
echo -e "${BLUE}Generating SDK...${NC}"
eval $CMD

# Show result
if [[ "$LANGUAGE" == "all" ]]; then
    echo ""
    echo -e "${GREEN}✓ SDKs generated for all languages!${NC}"
    echo ""
    echo -e "${BLUE}Generated SDKs:${NC}"
    for lang_dir in "$OUTPUT_DIR"/*/; do
        if [[ -d "$lang_dir/$SDK_NAME" ]]; then
            lang=$(basename "$lang_dir")
            echo -e "  ${GREEN}$lang${NC}: ${BLUE}$lang_dir$SDK_NAME${NC}"
            file_count=$(find "$lang_dir/$SDK_NAME" -type f 2>/dev/null | wc -l | tr -d ' ')
            echo -e "    ${YELLOW}($file_count files)${NC}"
        fi
    done
    echo ""
else
    SDK_PATH="$OUTPUT_DIR/$LANGUAGE/$SDK_NAME"
    if [[ -d "$SDK_PATH" ]]; then
        echo ""
        echo -e "${GREEN}✓ SDK generated successfully!${NC}"
        echo -e "Location: ${BLUE}$SDK_PATH${NC}"
        echo ""
        echo -e "${BLUE}Generated files:${NC}"
        find "$SDK_PATH" -type f | head -10
        echo ""
    fi
fi

