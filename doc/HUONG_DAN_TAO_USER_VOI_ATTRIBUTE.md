# Hướng Dẫn Tạo User Với Attribute (Fabric CA)

## Mục Đích
Tạo Admin cho từng trường với attribute `universityID` để chặn việc giả mạo bằng của trường khác.

## Bước 1: Tạo User Với Attribute

### 1.1. Tạo Admin Trường HUST
```bash
# Vào thư mục CA của Org1
cd ~/blockchain/network/organizations/peerOrganizations/org1.example.com

# Đăng ký user mới với attribute
export FABRIC_CA_CLIENT_HOME=${PWD}

fabric-ca-client register \
  --caname ca-org1 \
  --id.name admin_hust \
  --id.secret admin_hust_pw \
  --id.type client \
  --id.attrs 'universityID=UNI_HUST:ecert' \
  --tls.certfiles "${PWD}/tls-cert.pem"

# Enroll user
fabric-ca-client enroll \
  -u https://admin_hust:admin_hust_pw@localhost:7054 \
  --caname ca-org1 \
  -M "${PWD}/users/admin_hust@org1.example.com/msp" \
  --tls.certfiles "${PWD}/tls-cert.pem"

# Copy config
cp "${PWD}/msp/config.yaml" "${PWD}/users/admin_hust@org1.example.com/msp/config.yaml"
```

### 1.2. Tạo Admin Trường BKA
```bash
fabric-ca-client register \
  --caname ca-org1 \
  --id.name admin_bka \
  --id.secret admin_bka_pw \
  --id.type client \
  --id.attrs 'universityID=UNI_BKA:ecert' \
  --tls.certfiles "${PWD}/tls-cert.pem"

fabric-ca-client enroll \
  -u https://admin_bka:admin_bka_pw@localhost:7054 \
  --caname ca-org1 \
  -M "${PWD}/users/admin_bka@org1.example.com/msp" \
  --tls.certfiles "${PWD}/tls-cert.pem"

cp "${PWD}/msp/config.yaml" "${PWD}/users/admin_bka@org1.example.com/msp/config.yaml"
```

---

## Bước 2: Copy User Vào Container

```bash
cd ~/blockchain/network

# Copy Admin HUST
docker cp organizations/peerOrganizations/org1.example.com/users/admin_hust@org1.example.com/msp peer0.org1.example.com:/tmp/admin_hust_msp

# Copy Admin BKA
docker cp organizations/peerOrganizations/org1.example.com/users/admin_bka@org1.example.com/msp peer0.org1.example.com:/tmp/admin_bka_msp
```

---

## Bước 3: Test Phân Quyền

### Test 1: Admin HUST tạo bằng của HUST (Thành công)
```bash
docker exec -it peer0.org1.example.com sh
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_MSPCONFIGPATH=/tmp/admin_hust_msp
export CORE_PEER_TLS_ENABLED=true

peer chaincode invoke \
  -o orderer.example.com:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile /tmp/orderer.crt \
  -C certificatechannel -n certificate \
  --peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt \
  -c '{"function":"CreateCertificate","Args":["CERT_HUST_001","SV001","Nguyen Van A","CNTT","Gioi","2024","UNI_HUST"]}'
```
**Kết quả:** `status:200` ✅

### Test 2: Admin HUST cố tạo bằng của BKA (Thất bại)
```bash
# Vẫn dùng admin_hust_msp
peer chaincode invoke \
  -o orderer.example.com:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile /tmp/orderer.crt \
  -C certificatechannel -n certificate \
  --peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt \
  -c '{"function":"CreateCertificate","Args":["CERT_FAKE","SV002","Hacker","IT","Gioi","2024","UNI_BKA"]}'
```
**Kết quả:** `Error: bạn chỉ được tạo bằng của trường UNI_HUST, không được giả mạo trường UNI_BKA` ❌

### Test 3: Super Admin (Không có attribute) tạo bằng bất kỳ (Thành công)
```bash
export CORE_PEER_MSPCONFIGPATH=/tmp/admin_msp  # Admin gốc không có attribute

peer chaincode invoke \
  -o orderer.example.com:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile /tmp/orderer.crt \
  -C certificatechannel -n certificate \
  --peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt \
  -c '{"function":"CreateCertificate","Args":["CERT_ANY","SV003","Test","IT","Gioi","2024","UNI_ANY"]}'
```
**Kết quả:** `status:200` ✅ (Super Admin có quyền tạo bằng của mọi trường)

---

## Lưu Ý Quan Trọng

1. **Fabric CA phải đang chạy:** Kiểm tra bằng `docker ps | grep ca`
2. **Attribute phải có `:ecert`:** Để attribute được nhúng vào certificate
3. **Deploy lại chaincode:** Sau khi sửa code phải chạy `./deploy_network.sh`

---

## Tóm Tắt Cơ Chế Bảo Mật

| User | Attribute | Quyền |
|:-----|:----------|:------|
| `admin_hust` | `universityID=UNI_HUST` | Chỉ tạo bằng của HUST |
| `admin_bka` | `universityID=UNI_BKA` | Chỉ tạo bằng của BKA |
| `Admin` (gốc) | Không có | Tạo bằng của mọi trường (Super Admin) |
