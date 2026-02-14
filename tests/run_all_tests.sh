#!/bin/bash
# Run all Morty tests

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FAILED=0
PASSED=0
TOTAL=0

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Morty Test Suite${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

for test in "$SCRIPT_DIR"/test_*.sh; do
    TOTAL=$((TOTAL + 1))
    TEST_NAME=$(basename "$test")

    echo -e "${YELLOW}Running $TEST_NAME...${NC}"
    echo "========================================="

    if "$test"; then
        echo ""
        echo -e "${GREEN}✓ $TEST_NAME passed${NC}"
        PASSED=$((PASSED + 1))
    else
        echo ""
        echo -e "${RED}✗ $TEST_NAME failed${NC}"
        FAILED=$((FAILED + 1))
    fi
    echo ""
    echo ""
done

echo "========================================="
echo -e "${BLUE}Test Summary${NC}"
echo "========================================="
echo -e "Total tests:  $TOTAL"
echo -e "${GREEN}Passed:       $PASSED${NC}"

if [[ $FAILED -gt 0 ]]; then
    echo -e "${RED}Failed:       $FAILED${NC}"
    echo ""
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}Failed:       0${NC}"
    echo ""
    echo -e "${GREEN}All tests passed! ✨${NC}"
    exit 0
fi
