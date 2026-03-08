#!/bin/bash
set -e

# --- CONFIGURATION ---
CONTEXT_NAME="mint-server"
APP_NAME="landingpage"
DOMAIN="${APP_NAME}.my.os"
IMAGE_NAME="${APP_NAME}:latest"

echo "🚀 Deploying $APP_NAME using context $CONTEXT_NAME"

# 1️⃣ Verify Docker context
CURRENT_CONTEXT=$(docker context show)

if [ "$CURRENT_CONTEXT" != "$CONTEXT_NAME" ]; then
    echo "🔄 Switching to context $CONTEXT_NAME..."
    docker context use "$CONTEXT_NAME" || {
        echo "❌ Could not switch to context $CONTEXT_NAME"
        exit 1
    }
fi

echo "📡 Active context: $(docker context show)"

# 2️⃣ Build image
echo "🔨 Building image..."
docker build -t "$IMAGE_NAME" "$(dirname "$0")"

# 3️⃣ Remove previous container
echo "🧹 Cleaning up previous container..."
docker rm -f "$APP_NAME" 2>/dev/null || true

# 4️⃣ Run container
echo "🚀 Starting container..."

docker run -d \
    --name "$APP_NAME" \
    --network myos-net \
    --label "myproxy.domain=$DOMAIN" \
    --label "myproxy.fallback=true" \
    --label "myproxy.port=3000" \
    --env "PROXY_API_URL=http://myproxy:8080" \
    --restart unless-stopped \
    "$IMAGE_NAME"

echo "--------------------------------------"
echo "✅ $APP_NAME deployed successfully"
echo "✅ Access via: http://$DOMAIN"
echo "✅ Set as MyProxy fallback page"
echo "--------------------------------------"
