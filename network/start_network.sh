#!/bin/bash

# Bật lại mạng Blockchain — GIỮ NGUYÊN toàn bộ dữ liệu đã có
echo "🚀 Đang khởi động lại mạng Blockchain (giữ nguyên dữ liệu)..."

docker compose \
  -f docker/docker-compose-peer.yaml \
  -f docker/docker-compose-orderer.yaml \
  -f docker/docker-compose-couch.yaml \
  up -d

echo ""
echo "✅ Mạng Blockchain đã bật lại! Dữ liệu cũ vẫn còn nguyên."
echo "   Channel, Chaincode, và toàn bộ văn bằng trên ledger không bị ảnh hưởng."
