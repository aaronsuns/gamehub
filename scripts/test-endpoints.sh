#!/bin/bash
# Test all GameHub API endpoints
# Usage: ./scripts/test-endpoints.sh [base_url]
# Default: http://localhost:8080

BASE_URL="${1:-http://localhost:8080}"

echo "Testing GameHub API endpoints at $BASE_URL"
echo "=========================================="
echo ""

# Health check (no rate limit)
echo "1. Health check:"
echo "   GET $BASE_URL/health"
curl -s "$BASE_URL/health" | jq '.' 2>/dev/null || curl -s "$BASE_URL/health"
echo ""
echo ""

# Live series
echo "2. Live series:"
echo "   GET $BASE_URL/series/live"
curl -s "$BASE_URL/series/live" | jq 'length' 2>/dev/null && echo " items returned" || curl -s "$BASE_URL/series/live" | head -c 200 && echo "..."
echo ""
echo ""

# Live players
echo "3. Live players:"
echo "   GET $BASE_URL/players/live"
curl -s "$BASE_URL/players/live" | jq 'length' 2>/dev/null && echo " items returned" || curl -s "$BASE_URL/players/live" | head -c 200 && echo "..."
echo ""
echo ""

# Live teams
echo "4. Live teams:"
echo "   GET $BASE_URL/teams/live"
curl -s "$BASE_URL/teams/live" | jq 'length' 2>/dev/null && echo " items returned" || curl -s "$BASE_URL/teams/live" | head -c 200 && echo "..."
echo ""
echo ""

echo "=========================================="
echo "All endpoints tested!"
