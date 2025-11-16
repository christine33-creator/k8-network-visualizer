#!/bin/bash
# Quick Test - Network Flow Visualization
# Run this after ./test-flow-visualization.sh completes

set -e

echo "🧪 Quick Flow Visualization Test"
echo "================================"

# Setup port-forward in background
echo "📡 Setting up port-forward..."
kubectl -n network-visualizer port-forward svc/network-visualizer 8080:80 > /dev/null 2>&1 &
PF_PID=$!
sleep 3

echo ""
echo "✅ 1. Testing API Health"
curl -s http://localhost:8080/api/health | jq

echo ""
echo "✅ 2. Viewing Recent Flows (last 5)"
curl -s http://localhost:8080/api/flows?limit=5 | jq -r '.[] | "\(.source_pod) → \(.dest_pod) (\(.protocol)) [\(.verdict)]"'

echo ""
echo "✅ 3. Flow Metrics Summary"
curl -s http://localhost:8080/api/flows/metrics | jq -r '.[] | "  \(.source_id) → \(.dest_id)\n    Bandwidth: \(.bytes_per_sec | tostring) B/s\n    Packets: \(.packets_per_sec | tostring) pps\n    Connections: \(.connection_count)\n    Errors: \(.error_rate * 100 | tostring)%\n"'

echo ""
echo "✅ 4. Anomalies Detected"
ANOMALIES=$(curl -s http://localhost:8080/api/flows/anomalies)
COUNT=$(echo "$ANOMALIES" | jq '. | length')
echo "  Total: $COUNT anomalies"
if [ "$COUNT" -gt 0 ]; then
    echo "$ANOMALIES" | jq -r '.[] | "  [\(.severity | ascii_upcase)] \(.title)\n    \(.description)\n"' | head -20
fi

echo ""
echo "✅ 5. Active Flows"
curl -s http://localhost:8080/api/flows/active | jq -r '.[] | "  \(.source) → \(.target) (\(.flow_data.protocol))\n    \(.flow_data.bytes_per_sec) B/s, \(.flow_data.connection_count) connections"'

echo ""
echo "🎯 Generating Test Traffic Spike..."
kubectl -n test-apps exec traffic-generator -- bash -c 'for i in {1..30}; do curl -s http://backend-api-service:8080 > /dev/null 2>&1 & done; wait' > /dev/null 2>&1 || true

echo "⏳ Waiting 10 seconds for detection..."
sleep 10

echo ""
echo "✅ 6. New Anomalies After Traffic Spike"
curl -s http://localhost:8080/api/flows/anomalies | jq -r '.[] | select(.detected_at > (now - 20 | strftime("%Y-%m-%dT%H:%M:%SZ"))) | "  [\(.severity | ascii_upcase)] \(.title) (score: \(.score))\n    \(.description)"'

echo ""
echo "================================"
echo "🌐 Access UI: http://localhost:8080"
echo ""
echo "📊 What to look for in the UI:"
echo "  1. Active Flows badge in top bar"
echo "  2. Pulsing blue edges during traffic"
echo "  3. Varying edge widths (bandwidth)"
echo "  4. Click edges to see flow metrics"
echo "  5. Orange anomaly warnings"
echo ""
echo "🧹 Cleanup: kill $PF_PID"
echo ""
