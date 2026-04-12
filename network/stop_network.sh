#!/bin/bash

# Dừng các container của mạng Blockchain nhưng GIỮ NGUYÊN dữ liệu
echo "Đang tắt Blockchain (Peers, Orderer, CouchDB)..."
docker compose -f docker/docker-compose-peer.yaml \
-f docker/docker-compose-orderer.yaml \
-f docker/docker-compose-couch.yaml down --remove-orphans

echo "✅ Mạng Blockchain đã tắt! Dữ liệu vẫn còn nguyên."
echo "👉 Để bật lại: ./start_network.sh"
echo "👉 Để xóa sạch và làm lại từ đầu: ./deploy_network.sh"