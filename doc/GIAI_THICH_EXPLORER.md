# TỪ ĐIỂN KHÁI NIỆM TRONG HYPERLEDGER EXPLORER

Tài liệu này giải thích ý nghĩa các thông số kỹ thuật hiển thị trên Hyperledger Explorer, giúp bạn hiểu rõ "nhịp đập" của mạng Blockchain.

---

## 1. CÁC KHÁI NIỆM CƠ BẢN (Dashboard)

### **BLOCK (Khối)**
*   **Định nghĩa:** Là một "trang" trong cuốn sổ cái (Ledger).
*   **Ý nghĩa:** Mọi dữ liệu thay đổi đều được đóng gói vào Block. Blockchain là một chuỗi các Block nối tiếp nhau.
*   **Block Height (Chiều cao khối):** Chính là **tổng số lượng Block** hiện có. Block mới nhất luôn có số thứ tự cao nhất. (Ví dụ: Block 0, Block 1, ..., Block 100).

### **TRANSACTION / TX (Giao dịch)**
*   **Định nghĩa:** Là một hành động cụ thể (VD: Tạo bằng, Thu hồi bằng) được ghi lại trong Block.
*   **Quan hệ:** Một Block có thể chứa nhiều Transaction (tùy cấu hình, thường là 10-20 tx/block hoặc đóng block sau 2 giây).

### **NODE (Nút mạng)**
*   **Định nghĩa:** Các máy tính (server) tham gia vào mạng.
*   **Ví dụ:** `Peer0.Org1`, `Peer0.Org2`, `Orderer`.

### **CHAINCODE (Hợp đồng thông minh)**
*   **Định nghĩa:** Đoạn mã lập trình (Go/Java/Node) quy định logic nghiệp vụ (Ví dụ: hàm `CreateCertificate`).
*   **Ý nghĩa:** Là luật chơi chung mà tất cả các bên phải tuân thủ.

---

## 2. CHI TIẾT VỀ BLOCK (Block Details)

Khi bạn bấm vào một số của Block (ví dụ Block #5), bạn sẽ thấy:

| Thông số | Giải thích | Ví dụ đời thường |
| :--- | :--- | :--- |
| **Block Hash** | Mã băm duy nhất của khối này. Bất kỳ thay đổi nhỏ nào trong khối cũng làm mã này thay đổi hoàn toàn. | "Vân tay" của trang giấy. |
| **Data Hash** | Mã băm của toàn bộ dữ liệu (transactions) bên trong khối. | "Vân tay" của nội dung văn bản. |
| **Previous Hash** | Mã băm của khối liền trước nó. Đây là cái "xích" để nối các khối lại với nhau. | Số trang của trang trước. |
| **Number of Tx** | Số lượng giao dịch nằm trong khối này. | Số dòng nhật ký trong trang. |

---

## 3. CHI TIẾT VỀ GIAO DỊCH (Transaction Details)

Khi bấm vào một **TxID**, đây là phần quan trọng nhất để chứng minh độ minh bạch:

### **Transaction ID (TxID)**
Mã định danh duy nhất của giao dịch. Dùng để tra cứu, khiếu nại.

### **Creator (Người tạo)**
Danh tính (MSP ID) của người đã gửi giao dịch này.
*   *Ví dụ:* `Org1MSP` (Biết ngay là người của trường Org1 tạo ra).

### **Status (Trạng thái)**
*   **VALID:** Giao dịch hợp lệ, đã được ghi nhận.
*   **INVALID:** Giao dịch bị từ chối (có thể do sai chữ ký, hoặc dữ liệu bị sửa đổi cùng lúc - lỗi MVCC).

### **Read/Write Set (Quan trọng nhất)**
Đây là bằng chứng về việc dữ liệu đã thay đổi như thế nào:
*   **Read Set (Tập đọc):** Phiên bản dữ liệu mà Chaincode đã đọc trước khi xử lý (để đảm bảo không ai sửa chen ngang).
*   **Write Set (Tập ghi):** Cặp `Key-Value` mới sẽ được ghi đè xuống Database.
    *   *Ví dụ:* `Key: CERT001` -> `Value: {studentName: "Nguyen Van A", status: "valid"}`.

### **Endorser (Người xác thực)**
Danh sách các chữ ký của các Peer đã chấp thuận giao dịch này.
*   Nếu Policy là "OR", bạn sẽ thấy chữ ký của `Org1`.
*   Nếu Policy là "AND", bạn sẽ thấy chữ ký của cả `Org1` và `Org2`.

---

## 4. CÁC THÀNH PHẦN MẠNG (Network)

### **Peer (Máy trạm)**
*   Nơi lưu trữ sổ cái (Ledger) và Chaincode.
*   Nơi thực thi Transaction khi có yêu cầu.

### **Orderer (Máy điều phối)**
*   KHÔNG lưu trữ sổ cái, KHÔNG chạy Chaincode.
*   Nhiệm vụ duy nhất: Nhận transaction từ khắp nơi -> Sắp xếp theo thứ tự -> Đóng gói thành Block -> Gửi trả lại cho các Peer.
*   *Ví dụ:* Giống như người thư thư toà soạn, gom thư lại đóng thành kiện rồi gửi đi, không cần biết thư viết gì.

### **Channel (Kênh)**
*   Một đường truyền riêng tư. Chỉ những ai tham gia Channel mới có Ledgers (Sổ cái) này.
*   Một mạng Blockchain có thể có nhiều Channel (Ví dụ: Channel "Văn Bằng", Channel "Học Phí",...). Dữ liệu 2 kênh này không liên quan đến nhau.

---

## 5. TÓM TẮT QUY TRÌNH KHI BẠN ẤN "TẠO BẰNG"
Nhìn vào Explorer, bạn có thể kể lại câu chuyện sau:

1.  **Proposal:** Admin gửi yêu cầu "Tạo bằng cho SV A" đến **Peer**.
2.  **Endorsement:** Peer chạy thử Chaincode, thấy hợp lệ, ký tên xác nhận (Endorser Signature).
3.  **Submission:** Admin gửi yêu cầu kèm chữ ký đến **Orderer**.
4.  **Ordering:** Orderer đóng gói yêu cầu đó vào **Block #X**.
5.  **Validation:** Block #X được gửi về tất cả các Peer. Peer kiểm tra lại lần cuối (Valid/Invalid).
6.  **Commit:** Nếu Valid, dữ liệu (Write Set) được ghi vào CouchDB -> **Hiện lên Explorer**.
