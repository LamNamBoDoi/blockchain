SLIDE 1: GIỚI THIỆU ĐỀ TÀI

Tên đề tài: XÂY DỰNG MẠNG BLOCKCHAIN HYPERLEDGER FABRIC QUẢN LÝ VĂN BẰNG

Mục tiêu:
Xây dựng hệ thống cấp phát, lưu trữ và xác minh văn bằng điện tử.
Đảm bảo tính minh bạch, toàn vẹn dữ liệu và chống giả mạo.

Phạm vi nghiên cứu:
Mô phỏng mạng lưới gồm 2 tổ chức: Trường Đại học và Cơ quan Quản lý (Bộ GD&ĐT).

---

SLIDE 2: KIẾN TRÚC HỆ THỐNG (INFRASTRUCTURE)

Mô hình mạng: 2 Tổ chức (Organizations)
Org 1 (Issuer - Trường Đại học):
- Node Peer0: Lưu trữ sổ cái và thực thi giao dịch.
- CouchDB: Database lưu trữ trạng thái.
- Vai trò: Cấp phát và Quản lý văn bằng.

Org 2 (Auditor - Bộ GD&ĐT):
- Node Peer0: Lưu trữ bản sao sổ cái để đối chứng.
- CouchDB: Database đồng bộ.
- Vai trò: Giám sát và Thanh tra.

Thành phần khác:
- Orderer Service: 1 Node Raft (Sắp xếp và đóng gói Block).
- Network: Docker Containers.

---

SLIDE 3: THIẾT KẾ SMART CONTRACT (CHAINCODE)

Thông tin kỹ thuật:
- Ngôn ngữ lập trình: Golang.
- Tên Chaincode: Certificate.

Cấu trúc dữ liệu (Asset):
- CertificateID: Mã định danh bằng.
- StudentName: Tên sinh viên.
- Status: Trạng thái (Valid / Revoked).
- Issuer: Đơn vị cấp.

Các chức năng chính:
1. CreateCertificate: Cấp bằng mới (Kèm kiểm tra quyền Admin).
2. GetCertificate: Tra cứu thông tin bằng.
3. GetCertificatesByStudent: Tìm kiếm lịch sử bằng cấp của sinh viên.
4. RevokeCertificate: Thu hồi bằng (Đổi trạng thái, giữ nguyên lịch sử).

---

SLIDE 4: GIẢI PHÁP PHÂN QUYỀN (ACCESS CONTROL)

Cơ chế bảo mật:
Sử dụng MSPID (Membership Service Provider) để xác thực người gọi giao dịch.

Quy tắc nghiệp vụ:
- Quyền GHI (Write): Chỉ cho phép Org1 (Trường học) thực hiện cấp mới hoặc thu hồi.
- Quyền ĐỌC (Read): Tất cả thành viên trong mạng (Trường, Bộ, Sinh viên) đều được phép tra cứu.

Kết quả:
Ngăn chặn hoàn toàn việc can thiệp trái phép từ bên ngoài hoặc từ đơn vị giám sát vào dữ liệu gốc của nhà trường.

---

SLIDE 5: KỊCH BẢN DEMO 1 - CẤP BẰNG THÀNH CÔNG

Hành động:
Admin của Trường (Org1) gửi transaction CreateCertificate.

Quy trình xử lý:
1. Peer Org1 kiểm tra logic -> Ký xác nhận (Endorse).
2. Orderer nhận giao dịch -> Đóng gói vào Block mới.
3. Dữ liệu được đồng bộ xuống Ledger của cả 2 Org.

Kết quả:
Hệ thống trả về status: 200.
Dữ liệu hiển thị ngay lập tức trên Dashboard Explorer.

---

SLIDE 6: KỊCH BẢN DEMO 2 - KIỂM THỬ BẢO MẬT

Tình huống:
Sử dụng danh tính của Thanh tra (Org2) hoặc Sinh viên (User1) để cố tình gọi hàm Cấp bằng (Create).

Kết quả:
Hệ thống từ chối giao dịch.
Thông báo lỗi: "Error: You do not have permission".

Ý nghĩa:
Chứng minh Smart Contract hoạt động đúng logic bảo mật, ngăn chặn tối đa rủi ro làm giả bằng cấp từ nội bộ.

---

SLIDE 7: KỊCH BẢN DEMO 3 - THU HỒI & TRUY VẾT

Hành động:
Admin Trường phát hiện sai phạm và gọi hàm RevokeCertificate.

Kết quả trên Blockchain:
- Trạng thái bằng chuyển từ "valid" sang "revoked".
- Lịch sử giao dịch (History) lưu lại đầy đủ 2 mốc thời gian: [Thời điểm tạo] và [Thời điểm thu hồi].

Ý nghĩa:
Đảm bảo tính bất biến của lịch sử. Dữ liệu cũ không bị xóa mất mà chỉ được cập nhật trạng thái mới.

---

SLIDE 8: CÔNG CỤ QUẢN LÝ & GIÁM SÁT

Giao diện: Hyperledger Explorer

Các chỉ số theo dõi:
- Block Height: Độ dài chuỗi khối hiện tại.
- Transaction Count: Tổng số giao dịch đã thực hiện.
- Block Hash: Mã băm đảm bảo tính toàn vẹn của dữ liệu.

---

SLIDE 9: KẾT LUẬN

Kết quả đạt được:
- Xây dựng thành công mạng Blockchain Private hoàn chỉnh trên nền tảng Hyperledger Fabric.
- Đáp ứng đầy đủ quy trình nghiệp vụ: Cấp phát - Lưu trữ - Bảo mật - Truy vết.
- Hệ thống hoạt động ổn định với cơ sở dữ liệu CouchDB.

Hướng phát triển:
- Xây dựng Web App (Frontend) cho người dùng cuối.
- Tích hợp chữ ký số và mã QR Code để tra cứu nhanh.
