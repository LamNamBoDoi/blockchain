# HƯỚNG DẪN CẤU HÌNH MẠNG HYPERLEDGER FABRIC

## MỤC LỤC
1. [Tổng quan về Hyperledger Fabric](#1-tổng-quan)
2. [Các thành phần chính](#2-các-thành-phần-chính)
3. [Quy trình cấu hình mạng](#3-quy-trình-cấu-hình-mạng)
4. [Chi tiết các bước cấu hình](#4-chi-tiết-các-bước-cấu-hình)
5. [Các file cấu hình quan trọng](#5-các-file-cấu-hình-quan-trọng)
6. [Checklist triển khai](#6-checklist-triển-khai)

---

## 1. TỔNG QUAN

### 1.1. Hyperledger Fabric là gì?
Hyperledger Fabric là một nền tảng blockchain được phép (permissioned blockchain) được thiết kế cho doanh nghiệp, cung cấp:
- **Tính riêng tư**: Chỉ những thành viên được phép mới có thể tham gia mạng
- **Khả năng mở rộng**: Có thể xử lý nhiều giao dịch đồng thời
- **Tính module**: Cho phép tùy chỉnh các thành phần

### 1.2. Kiến trúc cơ bản
```
┌─────────────────────────────────────────────────┐
│              Ứng dụng Client                     │
└──────────────────┬──────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────┐
│           Peer Nodes (Endorsers)                 │
│  - Lưu trữ sổ cái (Ledger)                      │
│  - Thực thi Chaincode                           │
│  - Xác thực giao dịch                           │
└──────────────────┬──────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────┐
│         Ordering Service (Orderers)              │
│  - Sắp xếp giao dịch                            │
│  - Tạo block                                     │
│  - Phân phối block cho Peers                    │
└──────────────────┬──────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────┐
│    Certificate Authority (CA)                    │
│  - Quản lý danh tính                            │
│  - Cấp phát chứng chỉ số                        │
└─────────────────────────────────────────────────┘
```

---

## 2. CÁC THÀNH PHẦN CHÍNH

### 2.1. Organizations (Tổ chức)
- **Định nghĩa**: Một tập hợp các thực thể có cùng quyền lợi trong mạng
- **Vai trò**: Quản lý các peer nodes và users
- **Yêu cầu**: Mỗi organization cần:
  - Certificate Authority (CA) riêng
  - Ít nhất 1 peer node
  - MSP (Membership Service Provider) configuration

### 2.2. Peers
**Peer Node** là thành phần cốt lõi lưu trữ và duy trì sổ cái:

#### Các loại Peer:
- **Endorsing Peer**: Thực thi chaincode và ký xác nhận giao dịch
- **Committing Peer**: Xác thực và commit block vào ledger
- **Anchor Peer**: Điểm kết nối giữa các organization

#### Chức năng chính:
- Lưu trữ blockchain ledger
- Thực thi smart contracts (chaincode)
- Xác thực giao dịch
- Duy trì state database

### 2.3. Orderer (Ordering Service)
**Vai trò**: Sắp xếp giao dịch và tạo block

#### Các loại Orderer:
- **Solo**: Chỉ 1 orderer (dùng cho dev/test)
- **Kafka**: Sử dụng Apache Kafka (deprecated)
- **Raft**: Consensus thuật toán Raft (khuyên dùng cho production)

#### Chức năng:
- Nhận giao dịch từ clients
- Sắp xếp giao dịch theo thứ tự
- Tạo block từ các giao dịch
- Phân phối block cho tất cả peers

### 2.4. Certificate Authority (CA)
**Vai trò**: Quản lý danh tính và chứng chỉ số

#### Chức năng:
- Đăng ký (register) users và components
- Cấp phát (enroll) chứng chỉ X.509
- Quản lý danh tính trong organization
- Cung cấp crypto material (khóa công khai/riêng tư)

### 2.5. Channel
**Định nghĩa**: Kênh riêng tư cho một nhóm organizations giao tiếp

#### Đặc điểm:
- Mỗi channel có sổ cái riêng
- Chỉ members của channel mới thấy được dữ liệu
- Một peer có thể tham gia nhiều channels
- Chaincode được deploy trên từng channel

### 2.6. Chaincode (Smart Contract)
**Định nghĩa**: Logic nghiệp vụ được thực thi trên blockchain

#### Ngôn ngữ hỗ trợ:
- Go
- JavaScript/TypeScript
- Java

#### Chu kỳ sống:
1. Package
2. Install trên peers
3. Approve bởi organizations
4. Commit lên channel

---

## 3. QUY TRÌNH CẤU HÌNH MẠNG

### Sơ đồ tổng quan:
```
BƯỚC 1: Chuẩn bị môi trường
    ↓
BƯỚC 2: Tạo crypto materials (chứng chỉ, khóa)
    ↓
BƯỚC 3: Tạo genesis block và channel artifacts
    ↓
BƯỚC 4: Khởi động mạng (peers, orderers, CAs)
    ↓
BƯỚC 5: Tạo và tham gia channel
    ↓
BƯỚC 6: Deploy chaincode
    ↓
BƯỚC 7: Khởi tạo và tương tác với chaincode
```

---

## 4. CHI TIẾT CÁC BƯỚC CẤU HÌNH

### BƯỚC 1: Chuẩn bị Môi trường

#### 1.1. Cài đặt Prerequisites
**Cần cài đặt:**
- Docker và Docker Compose (v2.0+)
- Go (v1.19+) - nếu viết chaincode bằng Go
- Node.js (v16+) - nếu viết chaincode bằng JavaScript
- Git
- curl, wget, jq

#### 1.2. Tải Fabric Binaries và Docker Images
**Các công cụ cần thiết:**
- `cryptogen`: Tạo crypto materials
- `configtxgen`: Tạo genesis block và channel configuration
- `peer`: CLI tool để tương tác với peers
- `orderer`: Orderer node binary
- `fabric-ca-client`: Tương tác với CA

**Docker images cần thiết:**
- `hyperledger/fabric-peer`
- `hyperledger/fabric-orderer`
- `hyperledger/fabric-ca`
- `hyperledger/fabric-tools`
- `hyperledger/fabric-ccenv` (build chaincode)

#### 1.3. Cấu trúc thư mục đề xuất
```
network/
├── organizations/          # Crypto materials
│   ├── ordererOrganizations/
│   └── peerOrganizations/
├── system-genesis-block/   # Genesis block
├── channel-artifacts/      # Channel configuration
├── chaincode/             # Smart contracts
├── scripts/               # Automation scripts
├── docker/                # Docker compose files
└── config/               # Configuration files
    ├── configtx.yaml
    ├── crypto-config.yaml
    └── core.yaml
```

---

### BƯỚC 2: Tạo Crypto Materials

#### 2.1. Mục đích
Tạo chứng chỉ số và khóa mã hóa cho:
- Organizations (MSP)
- Peer nodes
- Orderer nodes
- Admin users
- Client users

#### 2.2. Hai phương pháp tạo crypto materials

**Phương pháp 1: Sử dụng cryptogen (đơn giản, cho dev/test)**
- Tạo file `crypto-config.yaml`
- Định nghĩa cấu trúc organizations
- Chạy lệnh `cryptogen generate`

**Phương pháp 2: Sử dụng Fabric CA (khuyên dùng cho production)**
- Khởi động CA server cho mỗi organization
- Register và enroll từng identity
- Linh hoạt và bảo mật hơn

#### 2.3. Nội dung file crypto-config.yaml
Cần định nghĩa:
- **OrdererOrgs**: Danh sách orderer organizations
  - Tên domain
  - Số lượng orderer nodes
  - Hostname cho mỗi orderer
  
- **PeerOrgs**: Danh sách peer organizations
  - Tên domain
  - Số lượng peers
  - Số lượng users
  - Enable NodeOUs (organizational units)

#### 2.4. Cấu trúc crypto materials được tạo
```
organizations/
├── ordererOrganizations/
│   └── example.com/
│       ├── msp/                    # MSP cho orderer org
│       ├── orderers/               # Crypto cho từng orderer
│       │   └── orderer.example.com/
│       │       ├── msp/
│       │       └── tls/
│       └── users/                  # Admin users
│           └── Admin@example.com/
│               └── msp/
└── peerOrganizations/
    └── org1.example.com/
        ├── msp/                    # MSP cho peer org
        ├── peers/                  # Crypto cho từng peer
        │   └── peer0.org1.example.com/
        │       ├── msp/
        │       └── tls/
        └── users/                  # Users và admins
            ├── Admin@org1.example.com/
            └── User1@org1.example.com/
```

---

### BƯỚC 3: Tạo Genesis Block và Channel Artifacts

#### 3.1. File configtx.yaml
**File cấu hình quan trọng nhất**, định nghĩa:

**Organizations**: Cấu hình MSP cho mỗi organization
- ID (MSPID)
- MSP directory path
- Policies (Readers, Writers, Admins, Endorsement)
- AnchorPeers

**Orderer**: Cấu hình ordering service
- OrdererType (Solo/Raft)
- Addresses của orderers
- BatchTimeout, BatchSize
- Raft consensus configuration
- Policies

**Application**: Cấu hình cho application channel
- Organizations tham gia
- Policies
- Capabilities

**Profiles**: Template cho genesis block và channels
- **OrdererGenesis**: Profile cho genesis block
- **ChannelProfile**: Profile cho application channel

#### 3.2. Tạo Genesis Block
**Mục đích**: Block đầu tiên của ordering service, khởi tạo system channel

**Chứa:**
- Cấu hình orderer
- Consortium definition
- Policies

**Lệnh sử dụng**: `configtxgen` với profile OrdererGenesis

#### 3.3. Tạo Channel Configuration Transaction
**Mục đích**: Tạo file cấu hình để tạo application channel

**Chứa:**
- Channel name
- Organizations tham gia
- Policies
- Anchor peer configuration

**Các artifacts cần tạo:**
- Channel creation transaction (.tx)
- Anchor peer update transactions (cho mỗi org)

---

### BƯỚC 4: Khởi động Mạng

#### 4.1. Chuẩn bị Docker Compose Files

**docker-compose-ca.yaml**: Khởi động Certificate Authorities
- CA container cho mỗi organization
- Expose ports (7054, 8054, ...)
- Mount volumes cho crypto materials
- Environment variables (CA admin credentials)

**docker-compose-orderer.yaml**: Khởi động Orderers
- Orderer containers
- Mount genesis block
- Mount crypto materials
- Expose ports (7050, 7053)
- Environment variables (MSP, TLS)

**docker-compose-peer.yaml**: Khởi động Peers
- Peer containers cho mỗi organization
- CouchDB/LevelDB containers (state database)
- Mount crypto materials
- Expose ports (7051, 9051, ...)
- Environment variables (MSP, TLS, chaincode, database)

#### 4.2. Thứ tự khởi động
1. **CA services** (nếu dùng Fabric CA)
2. **Orderer nodes**
3. **Peer nodes**
4. **State databases** (CouchDB)

#### 4.3. Kiểm tra mạng
- Kiểm tra container đang chạy
- Kiểm tra logs của từng container
- Verify kết nối giữa các nodes
- Test TLS connections

---

### BƯỚC 5: Tạo và Tham gia Channel

#### 5.1. Tạo Channel
**Bởi**: Organization admin
**Sử dụng**: Channel creation transaction đã tạo ở bước 3

**Quy trình:**
1. Sử dụng peer CLI với identity của admin
2. Tạo channel với orderer
3. Channel genesis block được sinh ra

#### 5.2. Peers tham gia Channel
**Mỗi peer cần:**
1. Fetch genesis block của channel từ orderer
2. Join channel bằng genesis block
3. Verify peer đã join thành công

**Lặp lại cho**: Tất cả peers muốn tham gia channel

#### 5.3. Update Anchor Peers
**Mục đích**: Cho phép organizations khám phá nhau

**Quy trình:**
- Mỗi organization update channel với anchor peer configuration
- Sử dụng anchor peer update transaction đã tạo
- Chỉ organization admin mới có thể update

#### 5.4. Verify Channel Setup
- Kiểm tra peer đã join channel
- Kiểm tra anchor peers được set
- Kiểm tra channel configuration

---

### BƯỚC 6: Deploy Chaincode

#### 6.1. Lifecycle của Chaincode (Fabric 2.x+)
**5 bước bắt buộc:**

**1. Package Chaincode**
- Đóng gói source code
- Tạo file .tar.gz
- Chứa metadata (type, path, label)

**2. Install Chaincode trên Peers**
- Install lên tất cả endorsing peers
- Mỗi peer trả về package ID
- Package ID dùng cho các bước tiếp theo

**3. Approve Chaincode Definition**
- Mỗi organization approve chaincode
- Phải có đủ số organizations theo policy
- Định nghĩa:
  - Chaincode name
  - Version
  - Sequence number
  - Endorsement policy
  - Init required hay không
  - Private data collections (nếu có)

**4. Check Commit Readiness**
- Kiểm tra đủ organizations đã approve chưa
- Verify trước khi commit

**5. Commit Chaincode Definition**
- Commit chaincode definition lên channel
- Sau bước này chaincode sẵn sàng sử dụng
- Nếu Init required, phải invoke Init function

#### 6.2. Endorsement Policy
**Định nghĩa**: Quy tắc về organizations nào phải endorse giao dịch

**Ví dụ:**
- "OR('Org1MSP.member', 'Org2MSP.member')": Chỉ cần 1 org
- "AND('Org1MSP.member', 'Org2MSP.member')": Cần cả 2 orgs
- "OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member')": Cần 2/3 orgs

#### 6.3. Khởi tạo Chaincode
**Nếu Init required:**
- Phải invoke Init function trước khi sử dụng
- Thường để khởi tạo state ban đầu

**Nếu không Init required:**
- Có thể invoke functions ngay lập tức

---

### BƯỚC 7: Tương tác với Chaincode

#### 7.1. Invoke Transaction
**Quy trình transaction flow:**

1. **Client** gửi transaction proposal đến endorsing peers
2. **Endorsing Peers**:
   - Thực thi chaincode (simulate)
   - Tạo read-write set
   - Ký kết response (endorsement)
   - Trả về cho client
3. **Client**:
   - Thu thập endorsements
   - Verify đủ endorsement policy
   - Gửi transaction + endorsements đến orderer
4. **Orderer**:
   - Sắp xếp transactions
   - Tạo block
   - Phân phối block cho peers
5. **Peers**:
   - Validate transactions trong block
   - Kiểm tra endorsement policy
   - Kiểm tra MVCC (version conflicts)
   - Commit valid transactions vào ledger
   - Update state database

#### 7.2. Query
**Hai loại query:**

**Query chaincode:**
- Đọc state từ peer's ledger
- Không tạo transaction
- Chỉ từ 1 peer (không cần consensus)
- Nhanh, nhưng có thể không đồng nhất

**Invoke với read-only function:**
- Qua transaction flow đầy đủ
- Chậm hơn nhưng đồng nhất
- Có endorsement

---

## 5. CÁC FILE CẤU HÌNH QUAN TRỌNG

### 5.1. crypto-config.yaml
**Mục đích**: Template tạo crypto materials với cryptogen

**Sections chính:**
```yaml
OrdererOrgs:
  - Name: Orderer
    Domain: example.com
    Specs:
      - Hostname: orderer
      - Hostname: orderer2  # Nếu có nhiều orderers

PeerOrganizations:
  - Name: Org1
    Domain: org1.example.com
    EnableNodeOUs: true
    Template:
      Count: 2              # Số peers
    Users:
      Count: 1              # Số users (ngoài Admin)
```

### 5.2. configtx.yaml
**Mục đích**: Template tạo genesis block và channel configurations

**Sections chính:**

**Organizations:**
```yaml
- &Org1
    Name: Org1MSP
    ID: Org1MSP
    MSPDir: path/to/msp
    Policies: ...
    AnchorPeers:
      - Host: peer0.org1.example.com
        Port: 7051
```

**Orderer:**
```yaml
Orderer: &OrdererDefaults
    OrdererType: etcdraft
    Addresses:
      - orderer.example.com:7050
    BatchTimeout: 2s
    BatchSize:
      MaxMessageCount: 10
      AbsoluteMaxBytes: 99 MB
      PreferredMaxBytes: 512 KB
    EtcdRaft:
      Consenters:
        - Host: orderer.example.com
          Port: 7050
          ClientTLSCert: path/to/tls/cert
          ServerTLSCert: path/to/tls/cert
```

**Profiles:**
```yaml
Profiles:
    OrdererGenesis:
        Orderer:
            <<: *OrdererDefaults
            Organizations:
                - *OrdererOrg
        Consortiums:
            SampleConsortium:
                Organizations:
                    - *Org1
                    - *Org2
    
    ChannelProfile:
        Consortium: SampleConsortium
        Application:
            Organizations:
                - *Org1
                - *Org2
```

### 5.3. core.yaml (Peer Configuration)
**Mục đích**: Cấu hình cho peer node

**Các cấu hình quan trọng:**
- Peer ID và địa chỉ
- TLS settings
- MSP configuration
- Gossip protocol settings
- Ledger configuration
- Chaincode settings
- State database (LevelDB/CouchDB)
- Logging levels

### 5.4. orderer.yaml
**Mục đích**: Cấu hình cho orderer node

**Các cấu hình quan trọng:**
- Orderer type
- Listening address
- TLS settings
- Genesis block
- Local MSP
- Raft consensus parameters (nếu dùng Raft)

### 5.5. docker-compose.yaml
**Mục đích**: Định nghĩa và khởi động các services

**Services chính:**
- CA containers
- Orderer containers
- Peer containers
- CouchDB containers
- CLI container (tools)

**Cấu hình cho mỗi service:**
- Image
- Ports
- Volumes
- Environment variables
- Networks
- Dependencies

---

## 6. CHECKLIST TRIỂN KHAI

### Phase 1: Chuẩn bị (Pre-deployment)
- [ ] Cài đặt tất cả prerequisites
- [ ] Tải Fabric binaries và Docker images
- [ ] Tạo cấu trúc thư mục
- [ ] Chuẩn bị file crypto-config.yaml
- [ ] Chuẩn bị file configtx.yaml
- [ ] Review và customize các file cấu hình

### Phase 2: Tạo Artifacts
- [ ] Tạo crypto materials (cryptogen hoặc CA)
- [ ] Verify crypto materials được tạo đúng
- [ ] Tạo genesis block
- [ ] Tạo channel creation transaction
- [ ] Tạo anchor peer update transactions
- [ ] Verify tất cả artifacts

### Phase 3: Khởi động Mạng
- [ ] Start CA services (nếu dùng Fabric CA)
- [ ] Verify CAs đang chạy
- [ ] Start orderer nodes
- [ ] Verify orderers đang chạy và kết nối
- [ ] Start peer nodes
- [ ] Verify peers đang chạy
- [ ] Start state databases
- [ ] Verify tất cả containers healthy
- [ ] Check logs cho errors

### Phase 4: Cấu hình Channel
- [ ] Tạo channel
- [ ] Verify channel được tạo
- [ ] Fetch genesis block cho mỗi peer
- [ ] Join tất cả peers vào channel
- [ ] Verify peers đã join thành công
- [ ] Update anchor peers cho mỗi organization
- [ ] Verify anchor peers được set

### Phase 5: Deploy Chaincode
- [ ] Chuẩn bị chaincode source code
- [ ] Package chaincode
- [ ] Install chaincode trên tất cả endorsing peers
- [ ] Verify installation và lưu package IDs
- [ ] Approve chaincode definition (mỗi org)
- [ ] Check commit readiness
- [ ] Commit chaincode definition
- [ ] Verify chaincode committed
- [ ] Initialize chaincode (nếu cần)
- [ ] Verify chaincode ready

### Phase 6: Testing
- [ ] Test invoke transactions
- [ ] Verify transactions được commit
- [ ] Test query functions
- [ ] Verify query results
- [ ] Test endorsement policy
- [ ] Test error scenarios
- [ ] Check all peer ledgers đồng bộ
- [ ] Monitor performance

### Phase 7: Monitoring & Maintenance
- [ ] Setup logging và monitoring
- [ ] Configure backup strategy
- [ ] Document network topology
- [ ] Document all configurations
- [ ] Create operational runbooks
- [ ] Setup alerts
- [ ] Plan upgrade strategy

---

## 7. BEST PRACTICES

### 7.1. Security
- **Luôn enable TLS** cho tất cả communications
- **Sử dụng Fabric CA** cho production thay vì cryptogen
- **Rotate certificates** định kỳ
- **Protect private keys** - không commit vào git
- **Use hardware security modules (HSM)** cho production
- **Implement proper access control** với policies
- **Audit logs** định kỳ

### 7.2. Performance
- **Sử dụng CouchDB** nếu cần rich queries
- **Optimize endorsement policy** - không yêu cầu quá nhiều endorsers
- **Tune batch size và timeout** cho orderer
- **Use private data collections** cho dữ liệu sensitive
- **Monitor resource usage** và scale khi cần
- **Optimize chaincode** - tránh loops lớn, expensive operations

### 7.3. High Availability
- **Multiple orderer nodes** với Raft consensus
- **Multiple peers** cho mỗi organization
- **Backup và disaster recovery** plan
- **Geographic distribution** của nodes
- **Load balancing** cho client requests

### 7.4. Development
- **Use version control** cho tất cả configs và chaincode
- **Separate environments** (dev, test, staging, prod)
- **Automated testing** cho chaincode
- **CI/CD pipeline** cho deployment
- **Document everything** - architecture, configs, procedures

### 7.5. Operations
- **Monitoring và alerting** system
- **Log aggregation** và analysis
- **Regular backups** của ledger và state
- **Patch management** strategy
- **Capacity planning**
- **Incident response** procedures

---

## 8. TÀI LIỆU THAM KHẢO

### Official Documentation
- Hyperledger Fabric Documentation: https://hyperledger-fabric.readthedocs.io/
- Fabric Samples: https://github.com/hyperledger/fabric-samples
- Fabric CA Documentation: https://hyperledger-fabric-ca.readthedocs.io/

### Community Resources
- Hyperledger Discord/Slack
- Stack Overflow (tag: hyperledger-fabric)
- GitHub Issues

### Training
- edX Hyperledger Courses
- Official Hyperledger Training

---

## 9. VẤN ĐỀ THƯỜNG GẶP

### 9.1. Connection Issues
**Vấn đề**: Peers không kết nối được với orderer
**Giải pháp**:
- Check network configuration trong Docker
- Verify TLS certificates
- Check orderer addresses trong config
- Review logs cho chi tiết errors

### 9.2. Chaincode Issues
**Vấn đề**: Chaincode install/approve fails
**Giải pháp**:
- Verify package format
- Check peer có đủ permissions
- Review chaincode dependencies
- Check logs của peer

### 9.3. Transaction Failures
**Vấn đề**: Transactions bị reject
**Giải pháp**:
- Verify endorsement policy được thỏa mãn
- Check chaincode logic cho errors
- Review MVCC conflicts
- Verify channel membership

### 9.4. Performance Issues
**Vấn đề**: Slow transaction processing
**Giải pháp**:
- Optimize chaincode
- Tune orderer batch settings
- Add more endorsing peers
- Review database performance (CouchDB/LevelDB)

---

## KẾT LUẬN

Cấu hình mạng Hyperledger Fabric là một quy trình phức tạp đòi hỏi hiểu biết sâu về:
- Kiến trúc blockchain
- Các thành phần của Fabric
- Security và cryptography
- Network và infrastructure
- Container orchestration

**Các điểm quan trọng cần nhớ:**
1. Luôn bắt đầu với test network đơn giản
2. Hiểu rõ từng thành phần trước khi deploy
3. Document tất cả configurations
4. Test kỹ lưỡng trước khi lên production
5. Plan cho security, scalability, và high availability
6. Monitor và maintain network định kỳ

Hãy follow checklist và best practices để đảm bảo network của bạn được cấu hình đúng cách và sẵn sàng cho production.
