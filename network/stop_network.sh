#!/bin/bash

# Dừng tất cả container blockchain (không cần biết file compose nào)
echo "Đang tắt Blockchain..."

docker stop $(docker ps --filter "name=peer" --filter "name=orderer" --filter "name=couchdb" -q) 2>/dev/null

echo "✅ Mạng Blockchain đã tắt! Container & dữ liệu vẫn còn nguyên."
echo "👉 Để bật lại: ./start_network.sh"
echo "👉 Để xóa sạch và làm lại từ đầu: ./deploy_network.sh"
