#!/bin/bash

# Script upgrade chaincode KHÔNG mất dữ liệu blockchain
# Chỉ upgrade chaincode version mới, giữ nguyên ledger và world state
#
# MÔI TRƯỜNG: WSL2 + Docker Desktop
# Tất cả peer CLI chạy bên trong Docker container để resolve được hostname Fabric
#
# Cách dùng:
#   cd blockchain/network
#   ./upgrade_chaincode.sh
#
# Mỗi lần upgrade: tăng CHAINCODE_VERSION và CHAINCODE_SEQUENCE

set -e

NETWORK_DIR=$(cd "$(dirname "$0")" && pwd)
FABRIC_TOOLS_IMAGE="hyperledger/fabric-tools:2.5.0"

# ─── VERSION CONFIG ────────────────────────────────────────────
# THAY ĐỔI MỖI LẦN UPGRADE
CHAINCODE_NAME="certificate"
CHAINCODE_VERSION="1.8"    # Tăng: 1.0 → 1.1 → 1.2 → 1.3 → 1.4 → 1.5 → 1.6 → 1.7 → 1.8 ...
CHAINCODE_SEQUENCE=9        # Tăng: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 ...
# ──────────────────────────────────────────────────────────────

CHANNEL_NAME="certificatechannel"
CHAINCODE_PATH="${NETWORK_DIR}/../chaincode/certificate"
SIGNATURE_POLICY="OR('Org1MSP.peer','Org2MSP.peer')"

echo "================ VERIFY NETWORK ================"
docker ps | grep peer || {
    echo "ERROR: Peers not running. Run deploy_network.sh first."
    exit 1
}

FABRIC_NET=$(docker network ls --format '{{.Name}}' --filter type=custom | while read n; do
    if docker network inspect "$n" 2>/dev/null | grep -q '"peer0.org1.example.com"'; then
        echo "$n"
    fi
done | head -n1)

if [ -z "${FABRIC_NET}" ]; then
    echo "ERROR: Could not auto-detect Fabric Docker network."
    docker inspect $(docker ps -q --filter name=peer) --format '{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}' | tr ' ' '\n' | sort -u
    exit 1
fi
echo "Docker network: ${FABRIC_NET} (auto-detected). Proceeding..."

echo "================ PACKAGE CHAINCODE v${CHAINCODE_VERSION} ================"

# Package: cần Go compiler, chạy trong WSL (không phải container)
export PATH=${NETWORK_DIR}/../bin:${PATH}
peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz \
    --path "${CHAINCODE_PATH}" \
    --lang golang \
    --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}

echo "================ CHECK INSTALLATION STATUS ================"

# Kiểm tra xem chaincode với version này đã được install chưa
EXISTING_PKG=$(docker run --rm --network "${FABRIC_NET}" \
    -v "${NETWORK_DIR}:/workspace" \
    -w /workspace \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_LOCALMSPID="Org1MSP" \
    -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
    -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
    -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
    "${FABRIC_TOOLS_IMAGE}" \
    peer lifecycle chaincode queryinstalled 2>/dev/null | \
    grep "${CHAINCODE_NAME}_${CHAINCODE_VERSION}" | \
    awk '{print $3}' | sed 's/,//')

if [ -n "${EXISTING_PKG}" ]; then
    echo "Chaincode v${CHAINCODE_VERSION} already installed on Org1. Skipping install."
    echo "Using existing PACKAGE_ID: ${EXISTING_PKG}"
    PACKAGE_ID="${EXISTING_PKG}"
else
    echo "Chaincode v${CHAINCODE_VERSION} not installed. Installing..."

    echo "================ INSTALL ON ORG1 ================"
    docker run --rm --network "${FABRIC_NET}" \
        -v "${NETWORK_DIR}:/workspace" \
        -w /workspace \
        -e CORE_PEER_TLS_ENABLED=true \
        -e CORE_PEER_LOCALMSPID="Org1MSP" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
        -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
        -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
        "${FABRIC_TOOLS_IMAGE}" \
        peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

    echo "================ INSTALL ON ORG2 ================"
    docker run --rm --network "${FABRIC_NET}" \
        -v "${NETWORK_DIR}:/workspace" \
        -w /workspace \
        -e CORE_PEER_TLS_ENABLED=true \
        -e CORE_PEER_LOCALMSPID="Org2MSP" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
        -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp \
        -e CORE_PEER_ADDRESS=peer0.org2.example.com:9051 \
        "${FABRIC_TOOLS_IMAGE}" \
        peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz

    echo "================ GET PACKAGE ID ================"
    PACKAGE_ID=$(docker run --rm --network "${FABRIC_NET}" \
        -v "${NETWORK_DIR}:/workspace" \
        -w /workspace \
        -e CORE_PEER_TLS_ENABLED=true \
        -e CORE_PEER_LOCALMSPID="Org1MSP" \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
        -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
        -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
        "${FABRIC_TOOLS_IMAGE}" \
        peer lifecycle chaincode queryinstalled 2>/dev/null | \
        grep "${CHAINCODE_NAME}_${CHAINCODE_VERSION}" | \
        awk '{print $3}' | sed 's/,//')
fi

echo "PACKAGE_ID = ${PACKAGE_ID}"

if [ -z "${PACKAGE_ID}" ]; then
    echo "ERROR: PACKAGE_ID is empty."
    exit 1
fi

echo "================ APPROVE ORG1 (sequence ${CHAINCODE_SEQUENCE}) ================"

docker run --rm --network "${FABRIC_NET}" \
    -v "${NETWORK_DIR}:/workspace" \
    -w /workspace \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_LOCALMSPID="Org1MSP" \
    -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
    -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
    -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
    "${FABRIC_TOOLS_IMAGE}" \
    peer lifecycle chaincode approveformyorg \
        -o orderer.example.com:7050 \
        --ordererTLSHostnameOverride orderer.example.com \
        --tls \
        --cafile /workspace/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
        --channelID "${CHANNEL_NAME}" \
        --name "${CHAINCODE_NAME}" \
        --version "${CHAINCODE_VERSION}" \
        --package-id "${PACKAGE_ID}" \
        --sequence "${CHAINCODE_SEQUENCE}" \
        --signature-policy "${SIGNATURE_POLICY}"

echo "================ APPROVE ORG2 (sequence ${CHAINCODE_SEQUENCE}) ================"

docker run --rm --network "${FABRIC_NET}" \
    -v "${NETWORK_DIR}:/workspace" \
    -w /workspace \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_LOCALMSPID="Org2MSP" \
    -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
    -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp \
    -e CORE_PEER_ADDRESS=peer0.org2.example.com:9051 \
    "${FABRIC_TOOLS_IMAGE}" \
    peer lifecycle chaincode approveformyorg \
        -o orderer.example.com:7050 \
        --ordererTLSHostnameOverride orderer.example.com \
        --tls \
        --cafile /workspace/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
        --channelID "${CHANNEL_NAME}" \
        --name "${CHAINCODE_NAME}" \
        --version "${CHAINCODE_VERSION}" \
        --package-id "${PACKAGE_ID}" \
        --sequence "${CHAINCODE_SEQUENCE}" \
        --signature-policy "${SIGNATURE_POLICY}"

echo "================ COMMIT CHAINCODE UPGRADE ================"

docker run --rm --network "${FABRIC_NET}" \
    -v "${NETWORK_DIR}:/workspace" \
    -w /workspace \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_LOCALMSPID="Org1MSP" \
    -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
    -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
    -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
    "${FABRIC_TOOLS_IMAGE}" \
    peer lifecycle chaincode commit \
        -o orderer.example.com:7050 \
        --ordererTLSHostnameOverride orderer.example.com \
        --tls \
        --cafile /workspace/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
        --channelID "${CHANNEL_NAME}" \
        --name "${CHAINCODE_NAME}" \
        --version "${CHAINCODE_VERSION}" \
        --sequence "${CHAINCODE_SEQUENCE}" \
        --signature-policy "${SIGNATURE_POLICY}" \
        --peerAddresses peer0.org1.example.com:7051 \
        --tlsRootCertFiles /workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
        --peerAddresses peer0.org2.example.com:9051 \
        --tlsRootCertFiles /workspace/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
        --connTimeout 60s

echo "================ VERIFY UPGRADE ================"

docker run --rm --network "${FABRIC_NET}" \
    -v "${NETWORK_DIR}:/workspace" \
    -w /workspace \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_LOCALMSPID="Org1MSP" \
    -e CORE_PEER_TLS_ROOTCERT_FILE=/workspace/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
    -e CORE_PEER_MSPCONFIGPATH=/workspace/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp \
    -e CORE_PEER_ADDRESS=peer0.org1.example.com:7051 \
    "${FABRIC_TOOLS_IMAGE}" \
    peer lifecycle chaincode querycommitted \
        --channelID "${CHANNEL_NAME}" \
        --name "${CHAINCODE_NAME}"

echo ""
echo "================  UPGRADE COMPLETE  ================"
echo "Chaincode upgraded to v${CHAINCODE_VERSION} (sequence ${CHAINCODE_SEQUENCE})"
echo "Blockchain data (ledgers, certificates) is PRESERVED"
echo ""
echo "Next upgrade: change CHAINCODE_VERSION=1.2 and CHAINCODE_SEQUENCE=3"
