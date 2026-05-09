#!/bin/sh
set -eu

api_url="${TXSIM_API_URL:-http://127.0.0.1:8080}"
escaped_api_url="$(printf '%s' "$api_url" | sed 's/\\/\\\\/g; s/"/\\"/g')"

cat > /usr/share/nginx/html/config.js <<EOF
window.__TXSIM_CONFIG__ = {
  apiUrl: "${escaped_api_url}"
};
EOF
