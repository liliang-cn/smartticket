#!/bin/bash

# SmartTicket OpenAPI Generation Script
# Generates OpenAPI documentation from proto files using protoc-gen-openapiv2

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Generating OpenAPI documentation from proto files...${NC}"

# Check if required tools are installed
check_tool() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}❌ $1 is not installed. Please install it first.${NC}"
        echo -e "${YELLOW}💡 Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest${NC}"
        exit 1
    fi
}

check_tool protoc
check_tool protoc-gen-go
check_tool protoc-gen-go-grpc
check_tool protoc-gen-openapiv2

# Set up directories
PROTO_DIR="./proto"
OUTPUT_DIR="./api"
TEMP_DIR="./tmp/proto"

# Create output directory
mkdir -p $OUTPUT_DIR
mkdir -p $TEMP_DIR

# Copy proto files to temp directory for processing
echo -e "${YELLOW}📁 Preparing proto files...${NC}"
cp -r $PROTO_DIR/* $TEMP_DIR/

# Generate OpenAPI documentation from all proto files
echo -e "${YELLOW}📝 Generating OpenAPI documentation from all proto files...${NC}"

# Find all proto files excluding test files
PROTO_FILES=$(find $TEMP_DIR -name "*.proto" -type f | grep -v "simple_api")

protoc \
    --openapiv2_out=$OUTPUT_DIR \
    --openapiv2_opt=logtostderr=true,json_names_for_fields=true,use_go_templates=true \
    -I $TEMP_DIR \
    $PROTO_FILES

# Check if OpenAPI files were generated
if [ -f "$OUTPUT_DIR/proto/smartticket/smartticket.swagger.json" ]; then
    echo -e "${GREEN}✅ OpenAPI JSON generated successfully: $OUTPUT_DIR/proto/smartticket/smartticket.swagger.json${NC}"
else
    echo -e "${RED}❌ Failed to generate OpenAPI documentation${NC}"
    exit 1
fi

# Copy generated files to main api directory for easier access
cp $OUTPUT_DIR/proto/smartticket/smartticket.swagger.json $OUTPUT_DIR/smartticket.v1.openapi.json

# Generate YAML version if yq is available
if command -v yq &> /dev/null; then
    echo -e "${YELLOW}📄 Converting to YAML format...${NC}"
    yq -P '.' $OUTPUT_DIR/smartticket.v1.openapi.json > $OUTPUT_DIR/smartticket.v1.openapi.yaml
    echo -e "${GREEN}✅ OpenAPI YAML generated: $OUTPUT_DIR/smartticket.v1.openapi.yaml${NC}"
else
    echo -e "${YELLOW}⚠️  yq not found, skipping YAML conversion${NC}"
fi

# Clean up temporary files
rm -rf $TEMP_DIR

echo -e "${GREEN}🎉 OpenAPI documentation generation complete!${NC}"
echo -e "${YELLOW}📚 Generated files:${NC}"
echo -e "   📄 $OUTPUT_DIR/smartticket.v1.openapi.json"
echo -e "   📄 $OUTPUT_DIR/proto/smartticket/smartticket.swagger.json"
if [ -f "$OUTPUT_DIR/smartticket.v1.openapi.yaml" ]; then
    echo -e "   📄 $OUTPUT_DIR/smartticket.v1.openapi.yaml"
fi

echo -e "${YELLOW}🔧 Next steps:${NC}"
echo -e "   1. Integrate with Swagger UI: https://swagger.io/tools/swagger-ui/"
echo -e "   2. Configure gRPC-Gateway for HTTP/REST endpoints"
echo -e "   3. Set up API authentication and middleware"