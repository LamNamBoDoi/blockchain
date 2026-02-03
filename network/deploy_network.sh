#!/bin/bash
# Script deploy mạng Blockchain và Chaincode tự động
# Cập nhật: Commit phải gửi cho cả Org1 và Org2. Policy OR chỉ áp dụng cho invoke sau này.

set -e

export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/config
export CORE_PEER_TLS_ENABLED=true
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
export PEER0_ORG1_CA=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export PEER0_ORG2_CA=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

CHANNEL_NAME="certificatechannel"
CHAINCODE_NAME="certificate"
CHAINCODE_VERSION="1.0"
CHAINCODE_SEQUENCE=1
CHAINCODE_PATH="../chaincode/certificate"
# Policy: Chỉ cần Org1 HOẶC Org2 ký là hợp lệ cho các transaction sau này
SIGNATURE_POLICY="OR('Org1MSP.peer','Org2MSP.peer')"

echo "=== 1. Dọn dẹp mạng cũ ==="
docker compose -f docker/docker-compose-peer.yaml -f docker/docker-compose-orderer.yaml -f docker/docker-compose-couch.yaml down --volumes --remove-orphans
docker volume prune -f

echo "=== 2. Khởi động mạng ==="
docker compose -f docker/docker-compose-peer.yaml -f docker/docker-compose-orderer.yaml -f docker/docker-compose-couch.yaml up -d
sleep 15

echo "=== 3. Tạo Channel ==="
configtxgen -profile ChannelProfile -outputCreateChannelTx ./channel-artifacts/${CHANNEL_NAME}.tx -channelID $CHANNEL_NAME

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer channel create -o localhost:7050 -c $CHANNEL_NAME -f ./channel-artifacts/${CHANNEL_NAME}.tx --outputBlock ./channel-artifacts/${CHANNEL_NAME}.block --tls --cafile $ORDERER_CA --ordererTLSHostnameOverride orderer.example.com

echo "=== 4. Join Channel ==="
echo "--- Org1 Join ---"
peer channel join -b ./channel-artifacts/${CHANNEL_NAME}.block

echo "--- Org2 Join ---"
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org2.example.com
peer channel join -b ./channel-artifacts/${CHANNEL_NAME}.block

echo "=== 5. Package & Install Chaincode ==="
# Org1 Install
echo "--- Org1 Install ---"
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz --path ${CHAINCODE_PATH} --lang golang --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}
peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

# Org2 Install
echo "--- Org2 Install ---"
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org2.example.com

peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

echo "=== 6. Approve Chaincode (With Policy) ==="
PACKAGE_ID=$(peer lifecycle chaincode queryinstalled -O json | jq -r ".installed_chaincodes | .[] | select(.package_id | startswith(\"${CHAINCODE_NAME}_${CHAINCODE_VERSION}\")) | .package_id" | head -n 1)
echo "Package ID: $PACKAGE_ID"

# Org1 Approve
echo "--- Org1 Approve ---"
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --package-id $PACKAGE_ID --sequence ${CHAINCODE_SEQUENCE} --signature-policy $SIGNATURE_POLICY

# Org2 Approve
echo "--- Org2 Approve ---"
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org2.example.com

peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA --channelID $CHANNEL_NAME --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --package-id $PACKAGE_ID --sequence ${CHAINCODE_SEQUENCE} --signature-policy $SIGNATURE_POLICY

echo "=== 7. Commit Chaincode (Inside Container) ==="
# Copy certificates into container
echo "Copying certs to peer0.org1..."
docker cp ${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp peer0.org1.example.com:/tmp/admin_msp
docker cp $ORDERER_CA peer0.org1.example.com:/tmp/orderer.crt
docker cp $PEER0_ORG2_CA peer0.org1.example.com:/tmp/org2.crt

# Exec Commit from inside container - GỬI CHO CẢ 2 ORG
echo "Running commit..."
docker exec peer0.org1.example.com sh -c "
  export CORE_PEER_MSPCONFIGPATH=/tmp/admin_msp
  export CORE_PEER_TLS_ENABLED=true
  peer lifecycle chaincode commit -o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile /tmp/orderer.crt --channelID $CHANNEL_NAME --name $CHAINCODE_NAME --version $CHAINCODE_VERSION --sequence $CHAINCODE_SEQUENCE --signature-policy \"$SIGNATURE_POLICY\" --peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles /etc/hyperledger/fabric/tls/ca.crt --peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles /tmp/org2.crt
"

echo "=== HOÀN TẤT ==="
docker ps
