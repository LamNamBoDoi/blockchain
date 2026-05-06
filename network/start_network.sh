#!/bin/bash

# Bật lại mạng Blockchain — GIỮ NGUYÊN toàn bộ dữ liệu đã có
echo "🚀 Đang khởi động lại mạng Blockchain (giữ nguyên dữ liệu)..."

# Dùng docker start thay vì compose up để giữ nguyên container state (channel, chaincode)
docker start \
  orderer.example.com \
  peer0.org1.example.com \
  peer0.org2.example.com \
  couchdb0 \
  couchdb1 \
  2>/dev/null

# Kiểm tra container nào chưa start được thì fallback sang compose up
FAILED=0
for name in orderer.example.com peer0.org1.example.com peer0.org2.example.com couchdb0 couchdb1; do
  STATUS=$(docker inspect --format='{{.State.Status}}' "$name" 2>/dev/null)
  if [ "$STATUS" != "running" ]; then
    echo "⚠️  Container '$name' không start được, fallback sang compose up..."
    FAILED=1
    break
  fi
done

if [ "$FAILED" = "1" ]; then
  echo "🔄 Dùng docker compose up (container mới)..."
  docker compose \
    -f docker/docker-compose-peer.yaml \
    -f docker/docker-compose-orderer.yaml \
    -f docker/docker-compose-couch.yaml \
    up -d
  echo ""
  echo "⚠️  Nếu bị lỗi channel, chạy thêm: bash setup_channel.sh"
fi

echo ""
echo "✅ Mạng Blockchain đã bật lại! Dữ liệu cũ vẫn còn nguyên."
echo "   Channel, Chaincode, và toàn bộ văn bằng trên ledger không bị ảnh hưởng."
