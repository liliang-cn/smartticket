#!/bin/bash

# SmartTicket API Documentation Cleanup Script
# Removes temporary files and ensures professional naming conventions

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧹 Cleaning up temporary files...${NC}"

# Remove any remaining temporary directories
if [ -d "./tmp" ]; then
    echo -e "${YELLOW}📁 Removing temporary directory...${NC}"
    rm -rf ./tmp
fi

# Remove any test files with "simple" in the name
find . -name "*simple*" -type f -exec rm {} \; 2>/dev/null || true

# Check for any remaining unprofessional file names
echo -e "${YELLOW}🔍 Checking for unprofessional file names...${NC}"

# List all files in proto directory that might be unprofessional
UNPROFESSIONAL_FILES=$(find proto/ -name "*simple*" -o -name "*test*" -o -name "*temp*" 2>/dev/null || true)

if [ -n "$UNPROFESSIONAL_FILES" ]; then
    echo -e "${RED}⚠️  Found potentially unprofessional files:${NC}"
    echo "$UNPROFESSIONAL_FILES"
    echo -e "${YELLOW}💡 Consider renaming these files to follow professional naming conventions${NC}"
else
    echo -e "${GREEN}✅ All file names follow professional conventions${NC}"
fi

# Verify final API directory structure
echo -e "${YELLOW}📋 Final API directory structure:${NC}"
echo -e "${GREEN}📄 api/${NC}"
echo -e "   ├─ 📄 smartticket.v1.openapi.json  # Main OpenAPI JSON specification"
echo -e "   ├─ 📄 openapi.yaml                  # OpenAPI 3.0 YAML specification"
echo -e "   ├─ 📄 swagger-ui.html               # Interactive Swagger UI"
echo -e "   ├─ 📄 README.md                     # Documentation"
echo -e "   └─ 📄 proto/smartticket/"
echo -e "       └─ 📄 smartticket.swagger.json  # Generated from proto"

# Check if all expected files exist
EXPECTED_FILES=(
    "api/smartticket.v1.openapi.json"
    "api/openapi.yaml"
    "api/swagger-ui.html"
    "api/README.md"
    "api/proto/smartticket/smartticket.swagger.json"
)

echo -e "${YELLOW}✅ Verifying expected files exist:${NC}"
for file in "${EXPECTED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "   ✅ $file"
    else
        echo -e "   ❌ $file (missing)"
    fi
done

echo -e "${GREEN}🎉 Cleanup complete!${NC}"