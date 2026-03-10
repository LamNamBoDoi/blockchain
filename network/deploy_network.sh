#!/bin/bash

set -e

# Tự tạo network fabric nếu chưa có
docker network create fabric 2>/dev/null || true

export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/config
export CORE_PEER_TLS_ENABLED=true

CHANNEL_NAME="certificatechannel"
CHAINCODE_NAME="certificate"
CHAINCODE_VERSION="1.0"
CHAINCODE_SEQUENCE=1
CHAINCODE_PATH="../chaincode/certificate"

SIGNATURE_POLICY="OR('Org1MSP.peer','Org2MSP.peer')"

ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
PEER0_ORG1_CA=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
PEER0_ORG2_CA=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

mkdir -p channel-artifacts

echo "================ CLEAN OLD NETWORK ================"

docker compose -f docker/docker-compose-peer.yaml \
-f docker/docker-compose-orderer.yaml \
-f docker/docker-compose-couch.yaml down --volumes --remove-orphans

docker volume prune -f

echo "================ GENERATE GENESIS BLOCK ================"
mkdir -p system-genesis-block
configtxgen -profile OrdererGenesis -channelID system-channel -outputBlock ./system-genesis-block/genesis.block -configPath ./config

echo "================ START NETWORK ================"

docker compose -f docker/docker-compose-peer.yaml \
-f docker/docker-compose-orderer.yaml \
-f docker/docker-compose-couch.yaml up -d

echo "Waiting for containers to start..."
sleep 30

docker ps

echo "================ CREATE CHANNEL TX ================"

configtxgen \
-profile ChannelProfile \
-outputCreateChannelTx ./channel-artifacts/${CHANNEL_NAME}.tx \
-channelID $CHANNEL_NAME

echo "================ CREATE CHANNEL ================"

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer channel create \
-o localhost:7050 \
-c $CHANNEL_NAME \
-f ./channel-artifacts/${CHANNEL_NAME}.tx \
--outputBlock ./channel-artifacts/${CHANNEL_NAME}.block \
--tls --cafile $ORDERER_CA \
--ordererTLSHostnameOverride orderer.example.com

echo "================ ORG1 JOIN CHANNEL ================"

peer channel join -b ./channel-artifacts/${CHANNEL_NAME}.block

echo "================ ORG2 JOIN CHANNEL ================"

export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org2.example.com

peer channel join -b ./channel-artifacts/${CHANNEL_NAME}.block

echo "================ PACKAGE CHAINCODE ================"

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz \
--path ${CHAINCODE_PATH} \
--lang golang \
--label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}

echo "================ INSTALL ORG1 ================"

peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

echo "================ INSTALL ORG2 ================"

export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org2.example.com

peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

echo "================ GET PACKAGE ID ================"

PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | grep ${CHAINCODE_NAME}_${CHAINCODE_VERSION} | awk '{print $3}' | sed 's/,//')

echo "PACKAGE_ID = $PACKAGE_ID"

echo "================ APPROVE ORG1 ================"

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer lifecycle chaincode approveformyorg \
-o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls --cafile $ORDERER_CA \
--channelID $CHANNEL_NAME \
--name $CHAINCODE_NAME \
--version $CHAINCODE_VERSION \
--package-id $PACKAGE_ID \
--sequence $CHAINCODE_SEQUENCE \
--signature-policy "$SIGNATURE_POLICY"

echo "================ APPROVE ORG2 ================"

export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG2_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org2.example.com

peer lifecycle chaincode approveformyorg \
-o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls --cafile $ORDERER_CA \
--channelID $CHANNEL_NAME \
--name $CHAINCODE_NAME \
--version $CHAINCODE_VERSION \
--package-id $PACKAGE_ID \
--sequence $CHAINCODE_SEQUENCE \
--signature-policy "$SIGNATURE_POLICY"

echo "================ COMMIT CHAINCODE ================"

docker run --rm --network fabric \
  -v ${PWD}:/workspace \
  -w /workspace \
  -e CORE_PEER_TLS_ENABLED=true \
  -e CORE_PEER_LOCALMSPID="Org1MSP" \
  -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
  -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
  -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
  hyperledger/fabric-tools:2.5.0 \
  peer lifecycle chaincode commit \
  -o orderer.example.com:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile /workspace/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  --channelID $CHANNEL_NAME \
  --name $CHAINCODE_NAME \
  --version $CHAINCODE_VERSION \
  --sequence $CHAINCODE_SEQUENCE \
  --signature-policy "$SIGNATURE_POLICY" \
  --peerAddresses peer0.org1.example.com:7051 \
  --tlsRootCertFiles /workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
  --peerAddresses peer0.org2.example.com:9051 \
  --tlsRootCertFiles /workspace/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

echo "================ VERIFY COMMIT ================"

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer lifecycle chaincode querycommitted \
--channelID $CHANNEL_NAME \
--name $CHAINCODE_NAME

echo "================ TEST INVOKE ================"

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PEER0_ORG1_CA
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_SERVERHOSTOVERRIDE=peer0.org1.example.com

peer chaincode invoke \
-o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls --cafile $ORDERER_CA \
-C $CHANNEL_NAME \
-n $CHAINCODE_NAME \
--peerAddresses localhost:7051 \
--tlsRootCertFiles $PEER0_ORG1_CA \
-c '{"function":"InitLedger","Args":[]}'

echo "================ DONE ================"

docker ps