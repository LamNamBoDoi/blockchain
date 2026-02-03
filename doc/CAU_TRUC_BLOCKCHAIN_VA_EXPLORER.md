# Cấu Trúc Blockchain & Hướng Dẫn Explorer

Tài liệu này giải thích sơ lược cấu trúc dữ liệu của Hyperledger Fabric và cách sử dụng công cụ để "nhìn thấy" blockchain trực quan.

## 1. Cấu Trúc Blockchain trong Hyperledger Fabric

Khác với Bitcoin hay Ethereum (một public blockchain đơn giản), Fabric là **Permissioned Blockchain** (có cấp phép) nên cấu trúc phức tạp hơn một chút:

### A. Sổ Cái (Ledger)
Mỗi Peer (nút mạng) lưu trữ một bản copy của Sổ cái. Sổ cái gồm 2 thành phần chính:
1.  **World State (Trạng thái hiện tại):**
    -   Lưu trữ giá trị *mới nhất* của các đối tượng (ví dụ: Asset1 đang thuộc về ai).
    -   Thường dùng **CouchDB** hoặc **LevelDB** để lưu. Điều này giúp truy vấn dữ liệu nhanh mà không cần quét lại toàn bộ lịch sử.
2.  **Blockchain (Chuỗi khối - Lịch sử):**
    -   Lưu trữ *nhật ký* tất cả các giao dịch đã xảy ra.
    -   Được lưu dưới dạng file trên ổ cứng của Peer.
    -   Không thể sửa đổi (Immutable).

### B. Block (Khối)
Một block trong Fabric bao gồm:
-   **Header**: Số thứ tự block (Block Number), Hash của block trước, Hash của dữ liệu hiện tại.
-   **Data**: Danh sách các giao dịch (RWSet - Read Write Set).
-   **Metadata**: Chữ ký của Orderer, thời gian tạo block, chứng nhận hợp lệ.

---

## 2. Làm sao xem được Blockchain? (Blockchain Explorer)

Bạn thắc mắc "xem bằng cái gì explore gì nhỉ", câu trả lời chính là: **Hyperledger Explorer**.

Đây là một công cụ giao diện web (Web UI) giúp bạn nhìn thấy những gì đang diễn ra bên trong mạng blockchain thay vì chỉ nhìn màn hình đen dòng lệnh.

### Hyperledger Explorer hiển thị những gì?
-   **Dashboard**: Tổng số blocks, transactions, số lượng nodes, chaincodes.
-   **Network**: Danh sách các Peer, Orderer đang tham gia.
-   **Blocks**: Xem chi tiết từng block (Block 0, Block 1...), ai tạo ra, chứa bao nhiêu giao dịch.
-   **Transactions**: Xem chi tiết từng ID giao dịch, ai gửi, gửi cho ai, dữ liệu thay đổi là gì.
-   **Chaincodes**: Xem mã nguồn hoặc thông tin chaincode đã deploy.

### Cách cài đặt Hyperledger Explorer

Explorer thường được chạy dưới dạng một Docker Container kết nối tới mạng Fabric của bạn.

**File cấu hình cơ bản (`docker-compose-explorer.yaml`):**

```yaml
version: '2.1'

services:
  explorer-db:
    image: hyperledger/explorer-db
    container_name: explorer-db
    environment:
      - DATABASE_DATABASE=fabricexplorer
      - DATABASE_USERNAME=hppoc
      - DATABASE_PASSWORD=password
    volumes:
      - ./pgdata:/var/lib/postgresql/data

  explorer:
    image: hyperledger/explorer
    container_name: explorer
    environment:
      - DATABASE_HOST=explorer-db
      - DATABASE_USERNAME=hppoc
      - DATABASE_PASSWORD=password
      - DISCOVERY_AS_LOCALHOST=false
    volumes:
      - ./config.json:/opt/explorer/app/platform/fabric/config.json
      - ./organizations:/tmp/crypto
    ports:
      - "9090:9090"
    depends_on:
      - explorer-db
```

Sau khi chạy `docker-compose up`, bạn truy cập trình duyệt tại `http://localhost:9090` (đăng nhập mặc định admin/adminpw) để xem giao diện quản trị.

### Các cách xem khác (Đơn giản hơn)

Nếu chưa cài Explorer, bạn vẫn có thể xem được dữ liệu thô:

1.  **Xem CouchDB (Nếu dùng CouchDB):**
    -   Truy cập `http://localhost:5984/_utils`.
    -   Tại đây bạn sẽ thấy giao diện quản lý DB (Fauxton). Bạn có thể xem trực tiếp `World State` (dữ liệu hiện tại) của các asset dưới dạng JSON.

2.  **Dùng lệnh Command Line:**
    -   Xem thông tin channel hiện tại (cao nhất là block bao nhiêu):
        ```bash
        peer channel getinfo -c mychannel
        ```
    -   Lấy block về để soi (kết quả là file protobuf khó đọc, cần giải mã):
        ```bash
        peer channel fetch 5 myblock.block -c mychannel
        ```
