# Certificate Chaincode (Go)

## Cấu trúc
```
blockchain/chaincode/certificate/
├── go.mod              # Go module dependencies
├── certificate.go      # Main chaincode logic
└── README.md          # This file
```

## Functions

### 1. InitLedger
Khởi tạo ledger (chạy 1 lần khi deploy)

### 2. CreateCertificate
Tạo certificate mới
```
Parameters:
- certID: ID của certificate
- studentID: ID sinh viên
- studentName: Tên sinh viên
- courseName: Tên khóa học
- grade: Điểm/Xếp loại
- issueDate: Ngày cấp
- issuer: Đơn vị cấp
```

### 3. GetCertificate
Lấy thông tin certificate theo ID

### 4. VerifyCertificate
Kiểm tra certificate có hợp lệ không

### 5. RevokeCertificate
Thu hồi certificate

### 6. GetAllCertificates
Lấy tất cả certificates

### 7. GetCertificateHistory
Lấy lịch sử thay đổi của certificate

### 8. GetCertificatesByStudent
Lấy tất cả certificates của 1 sinh viên

## Build & Deploy

### 1. Install dependencies
```bash
cd blockchain/chaincode/certificate
go mod tidy
```

### 2. Test compile
```bash
go build
```

### 3. Deploy to Fabric (từ test-network)
```bash
cd ../../fabric-samples/test-network

# Package
peer lifecycle chaincode package certificate.tar.gz \
  --path ../../chaincode/certificate \
  --lang golang \
  --label certificate_1.0

# Install on Org1
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
peer lifecycle chaincode install certificate.tar.gz

# Install on Org2
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
peer lifecycle chaincode install certificate.tar.gz

# Approve & Commit (xem DEPLOY.md)
```

## Test chaincode

### Invoke CreateCertificate
```bash
peer chaincode invoke \
  -o localhost:7050 \
  -C mychannel \
  -n certificate \
  -c '{"function":"CreateCertificate","Args":["CERT001","SV001","Nguyen Van A","Blockchain Development","A","2024-01-09","HCMUT"]}'
```

### Query GetCertificate
```bash
peer chaincode query \
  -C mychannel \
  -n certificate \
  -c '{"function":"GetCertificate","Args":["CERT001"]}'
```

### Verify Certificate
```bash
peer chaincode query \
  -C mychannel \
  -n certificate \
  -c '{"function":"VerifyCertificate","Args":["CERT001"]}'
```
