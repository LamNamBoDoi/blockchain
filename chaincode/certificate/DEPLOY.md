# Deploy Certificate Chaincode - Step by Step

## Prerequisites
- Fabric test network đang chạy
- Đã tạo channel `mychannel`

## Bước 1: Kiểm tra Go version
```bash
go version
# Cần Go 1.20 trở lên
```

## Bước 2: Install dependencies
```bash
cd blockchain/chaincode/certificate
go mod tidy
go mod vendor  # Tạo vendor folder
```

## Bước 3: Test compile
```bash
go build
# Nếu không lỗi → OK
```

## Bước 4: Package chaincode
```bash
cd ../../fabric-samples/test-network

peer lifecycle chaincode package certificate.tar.gz \
  --path ../../chaincode/certificate \
  --lang golang \
  --label certificate_1.0
```

## Bước 5: Install trên Org1
```bash
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode install certificate.tar.gz
```

**Lưu lại Package ID:**
```bash
peer lifecycle chaincode queryinstalled
# Output: certificate_1.0:xxxxx... → Copy Package ID này
```

## Bước 6: Install trên Org2
```bash
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer lifecycle chaincode install certificate.tar.gz
```

## Bước 7: Approve chaincode cho Org1
```bash
export CC_PACKAGE_ID=certificate_1.0:xxxxx  # Thay bằng Package ID từ bước 5

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode approveformyorg \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  --channelID mychannel \
  --name certificate \
  --version 1.0 \
  --package-id $CC_PACKAGE_ID \
  --sequence 1
```

## Bước 8: Approve chaincode cho Org2
```bash
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_ADDRESS=localhost:9051

peer lifecycle chaincode approveformyorg \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  --channelID mychannel \
  --name certificate \
  --version 1.0 \
  --package-id $CC_PACKAGE_ID \
  --sequence 1
```

## Bước 9: Check commit readiness
```bash
peer lifecycle chaincode checkcommitreadiness \
  --channelID mychannel \
  --name certificate \
  --version 1.0 \
  --sequence 1 \
  --output json
```

**Phải thấy:**
```json
{
  "approvals": {
    "Org1MSP": true,
    "Org2MSP": true
  }
}
```

## Bước 10: Commit chaincode
```bash
peer lifecycle chaincode commit \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  --channelID mychannel \
  --name certificate \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
  --peerAddresses localhost:9051 \
  --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
  --version 1.0 \
  --sequence 1
```

## Bước 11: Verify deployment
```bash
peer lifecycle chaincode querycommitted --channelID mychannel --name certificate
```

## Bước 12: Test chaincode

### Init Ledger
```bash
peer chaincode invoke \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  -C mychannel \
  -n certificate \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
  --peerAddresses localhost:9051 \
  --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
  -c '{"function":"InitLedger","Args":[]}'
```

### Create Certificate
```bash
peer chaincode invoke \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  -C mychannel \
  -n certificate \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
  --peerAddresses localhost:9051 \
  --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
  -c '{"function":"CreateCertificate","Args":["CERT001","SV001","Nguyen Van A","Blockchain Development","A","2024-01-09","HCMUT"]}'
```

### Query Certificate
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

## ✅ Hoàn thành!

Chaincode đã được deploy và sẵn sàng sử dụng.

## Troubleshooting

### Lỗi "chaincode not found"
```bash
# Kiểm tra chaincode đã commit chưa
peer lifecycle chaincode querycommitted --channelID mychannel
```

### Lỗi "package not found"
```bash
# Kiểm tra package đã install chưa
peer lifecycle chaincode queryinstalled
```

### Lỗi compile Go
```bash
# Xóa vendor và rebuild
rm -rf vendor
go mod tidy
go mod vendor
```
