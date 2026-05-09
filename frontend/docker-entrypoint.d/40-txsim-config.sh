#!/bin/sh
set -eu

api_url="${TXSIM_API_URL:-http://127.0.0.1:8080}"
single_line_api_url="$(printf '%s' "$api_url" | tr -d '\r\n')"
if [ "$single_line_api_url" != "$api_url" ]; then
  echo "TXSIM_API_URL must not contain carriage returns or newlines" >&2
  exit 1
fi

escaped_api_url="$(printf '%s' "$api_url" | sed 's/\\/\\\\/g; s/"/\\"/g')"

cat > /usr/share/nginx/html/config.js <<EOF
window.__TXSIM_CONFIG__ = {
  apiUrl: "${escaped_api_url}"
};
EOF
