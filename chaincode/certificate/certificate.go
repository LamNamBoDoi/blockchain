package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CertificateContract - Hợp đồng thông minh quản lý văn bằng
//
// Phân quyền tổ chức:
//
//	Org1MSP = Trường Đại Học
//	          → University Admin: tạo bằng, thu hồi bằng do trường mình cấp
//	          → Backend dùng identity Org1 để gọi chaincode thay mặt University Admin
//	Org2MSP = Bộ GD&ĐT / Super Admin
//	          → Super Admin: giám sát toàn hệ thống, thu hồi bất kỳ bằng nào
//	          → Backend dùng identity Org2 để revoke thay mặt Super Admin
type CertificateContract struct {
	contractapi.Contract
}

// Certificate - Cấu trúc dữ liệu văn bằng
type Certificate struct {
	CertID           string    `json:"certId"`           // Mã văn bằng (VD: CERT-BK-2025-001)
	UniversityID     string    `json:"universityId"`     // Mã trường (VD: BACHKHOA, KHTN, UEH)
	UniversityName   string    `json:"universityName"`   // Tên trường đầy đủ
	DegreeType       string    `json:"degreeType"`       // Loại bằng: "Cử nhân", "Thạc sĩ", "Tiến sĩ"
	Major            string    `json:"major"`            // Chuyên ngành
	StudentID        string    `json:"studentId"`        // Mã sinh viên
	StudentName      string    `json:"studentName"`      // Tên sinh viên
	Grade            string    `json:"grade"`            // Xếp loại: "Xuất sắc", "Giỏi", "Khá", "Trung bình"
	GraduationYear   string    `json:"graduationYear"`   // Năm tốt nghiệp
	IssueDate        string    `json:"issueDate"`        // Ngày cấp (VD: "2025-06-15")
	Issuer           string    `json:"issuer"`           // Người ký (VD: Hiệu trưởng)
	// ── QUAN TRỌNG NHẤT: Hash của file PDF văn bằng gốc ──────────────────────
	// Đây là trái tim của hệ thống xác minh.
	// Khi ai đó nộp file PDF, hệ thống tính SHA-256 của file đó
	// rồi so sánh với giá trị này. Nếu khớp → file gốc, không khớp → file giả/bị sửa.
	// Format: "sha256:<hex_string>" (VD: "sha256:a3f9c2d1e8b4...")
	PdfHash          string    `json:"pdfHash"`          // SHA-256 hash của file PDF gốc
	Status           string    `json:"status"`           // Trạng thái: "valid" | "revoked"
	RevokeReason     string    `json:"revokeReason"`     // Lý do thu hồi (nếu có)
	UpdateCount      int       `json:"updateCount"`      // Số lần cập nhật PDF hash
	LastUpdateReason string    `json:"lastUpdateReason"` // Lý do cập nhật gần nhất
	CreatedAt        time.Time `json:"createdAt"`        // Thời gian tạo trên blockchain
	UpdatedAt        time.Time `json:"updatedAt"`        // Thời gian cập nhật gần nhất
}

// UpdateRecord - Bản ghi lịch sử cập nhật PDF hash
type UpdateRecord struct {
	TxID       string    `json:"txId"`       // ID giao dịch blockchain
	Timestamp  time.Time `json:"timestamp"`  // Thời gian cập nhật
	OldPdfHash string    `json:"oldPdfHash"` // Hash cũ trước khi cập nhật
	NewPdfHash string    `json:"newPdfHash"` // Hash mới sau khi cập nhật
	Reason     string    `json:"reason"`     // Lý do cập nhật
	UpdatedBy  string    `json:"updatedBy"`  // MSP ID của người thực hiện
}

// HistoryQueryResult - Kết quả truy vấn lịch sử giao dịch
type HistoryQueryResult struct {
	TxID      string      `json:"txId"`      // ID giao dịch blockchain
	Timestamp time.Time   `json:"timestamp"` // Thời gian giao dịch
	IsDelete  bool        `json:"isDelete"`  // Đã xóa hay chưa
	Value     Certificate `json:"value"`     // Nội dung tại thời điểm đó
}

type VerifyResult struct {
	IsValid      bool         `json:"isValid"`      // Văn bằng có hợp lệ không
	PdfHashMatch bool         `json:"pdfHashMatch"` // Hash PDF có khớp không
	Status       string       `json:"status"`       // "valid", "revoked", "not_found", "pdf_tampered"
	Message      string       `json:"message"`      // Thông báo mô tả
	Certificate  *Certificate `json:"certificate"`  // Thông tin văn bằng đầy đủ
}

// ============================================================
// KHỞI TẠO
// ============================================================

// InitLedger - Khởi tạo sổ cái (có thể thêm dữ liệu mẫu nếu cần)
func (c *CertificateContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("✅ Chaincode Văn Bằng đã được khởi tạo thành công")
	return nil
}

// ============================================================
// CRUD CƠ BẢN
// ============================================================

// CreateCertificate - Cấp văn bằng mới lên blockchain
// Chỉ Org1MSP (Khối Trường Đại Học) mới được gọi hàm này
// pdfHash: SHA-256 của file PDF văn bằng, tính ở backend trước khi gọi chaincode
//
//	Format: "sha256:<64_hex_chars>"  VD: "sha256:a3f9c2d1e8b4..."
func (c *CertificateContract) CreateCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
	universityID string,
	universityName string,
	degreeType string,
	major string,
	studentID string,
	studentName string,
	grade string,
	graduationYear string,
	issueDate string,
	issuer string,
	pdfHash string, // ← SHA-256 hash của file PDF gốc - BẮT BUỘC
) error {
	// --- Kiểm tra quyền: Chỉ Org1MSP (Trường ĐH) được phép cấp bằng ---
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("không thể lấy MSP ID người gọi: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("không có quyền cấp văn bằng: chỉ Org1MSP (Trường ĐH) được phép, bạn là: %s", clientMSPID)
	}

	// --- Validate input bắt buộc ---
	if strings.TrimSpace(certID) == "" || strings.TrimSpace(universityID) == "" ||
		strings.TrimSpace(studentID) == "" || strings.TrimSpace(studentName) == "" {
		return fmt.Errorf("certId, universityId, studentId, studentName không được để trống")
	}

	// --- Validate pdfHash ---
	if strings.TrimSpace(pdfHash) == "" {
		return fmt.Errorf("pdfHash không được để trống: cần tính SHA-256 của file PDF trước khi cấp bằng")
	}
	if !strings.HasPrefix(pdfHash, "sha256:") || len(pdfHash) != 71 { // "sha256:" + 64 hex chars
		return fmt.Errorf("pdfHash không đúng định dạng, cần 'sha256:<64_hex_chars>', nhận được: %s", pdfHash)
	}

	// --- Kiểm tra văn bằng đã tồn tại chưa ---
	exists, err := c.CertificateExists(ctx, certID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("văn bằng %s đã tồn tại trên blockchain", certID)
	}

	// --- Lấy timestamp từ transaction (đảm bảo tính nhất quán giữa các peer) ---
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("không thể lấy timestamp giao dịch: %v", err)
	}
	timestamp := time.Unix(txTimestamp.Seconds, 0) // Bỏ nanoseconds để đảm bảo determinism

	// --- Tạo đối tượng văn bằng ---
	certificate := Certificate{
		CertID:           certID,
		UniversityID:     universityID,
		UniversityName:   universityName,
		DegreeType:       degreeType,
		Major:            major,
		StudentID:        studentID,
		StudentName:      studentName,
		Grade:            grade,
		GraduationYear:   graduationYear,
		IssueDate:        issueDate,
		Issuer:           issuer,
		PdfHash:          pdfHash, // ← Khóa chống làm giả: hash của PDF gốc
		Status:           "valid",
		RevokeReason:     "",
		UpdateCount:      0,  // ← Số lần cập nhật ban đầu = 0
		LastUpdateReason: "", // ← Chưa có lý do cập nhật nào
		CreatedAt:        timestamp,
		UpdatedAt:        timestamp,
	}

	// --- Lưu vào ledger ---
	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		return fmt.Errorf("không thể serialize dữ liệu: %v", err)
	}

	return ctx.GetStub().PutState(certID, certificateJSON)
}

// GetCertificate - Lấy thông tin văn bằng theo ID
func (c *CertificateContract) GetCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (*Certificate, error) {
	certificateJSON, err := ctx.GetStub().GetState(certID)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc dữ liệu: %v", err)
	}
	if certificateJSON == nil {
		return nil, fmt.Errorf("văn bằng %s không tồn tại", certID)
	}

	var certificate Certificate
	err = json.Unmarshal(certificateJSON, &certificate)
	if err != nil {
		return nil, fmt.Errorf("không thể deserialize dữ liệu: %v", err)
	}

	return &certificate, nil
}

// CertificateExists - Kiểm tra văn bằng có tồn tại không
func (c *CertificateContract) CertificateExists(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (bool, error) {
	certificateJSON, err := ctx.GetStub().GetState(certID)
	if err != nil {
		return false, fmt.Errorf("lỗi khi đọc world state: %v", err)
	}
	return certificateJSON != nil, nil
}

// ============================================================
// XÁC MINH & THU HỒI
// ============================================================

// VerifyCertificate - Xác minh văn bằng CHỈ bằng certID (không kiểm tra file)
// Dùng khi tra cứu nhanh thông tin, chưa có file PDF
func (c *CertificateContract) VerifyCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (*VerifyResult, error) {
	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return &VerifyResult{
			IsValid: false,
			Status:  "not_found",
			Message: fmt.Sprintf("Không tìm thấy văn bằng với mã: %s", certID),
		}, nil
	}

	if certificate.Status == "revoked" {
		return &VerifyResult{
			IsValid:     false,
			Status:      "revoked",
			Message:     fmt.Sprintf("Văn bằng đã bị thu hồi. Lý do: %s", certificate.RevokeReason),
			Certificate: certificate,
		}, nil
	}

	return &VerifyResult{
		IsValid:     true,
		Status:      "valid",
		Message:     "Văn bằng hợp lệ. Lưu ý: chưa xác minh file PDF - dùng VerifyWithPdfHash để xác minh đầy đủ.",
		Certificate: certificate,
	}, nil
}

// VerifyWithPdfHash - Xác minh ĐẦY ĐỦ: kiểm tra cả trạng thái VÀ tính toàn vẹn file PDF
// Đây là hàm xác minh CHÍNH XÁC NHẤT - dùng khi người dùng upload file PDF lên
// pdfHash: SHA-256 của file PDF người dùng nộp lên, tính ở frontend/backend
func (c *CertificateContract) VerifyWithPdfHash(
	ctx contractapi.TransactionContextInterface,
	certID string,
	pdfHash string, // Hash của file PDF mà người dùng đang cầm
) (*VerifyResult, error) {
	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return &VerifyResult{
			IsValid: false,
			Status:  "not_found",
			Message: fmt.Sprintf("Không tìm thấy văn bằng với mã: %s", certID),
		}, nil
	}

	// --- Bước 1: Kiểm tra trạng thái thu hồi ---
	if certificate.Status == "revoked" {
		return &VerifyResult{
			IsValid:      false,
			PdfHashMatch: false,
			Status:       "revoked",
			Message:      fmt.Sprintf("Văn bằng đã bị thu hồi. Lý do: %s", certificate.RevokeReason),
			Certificate:  certificate,
		}, nil
	}

	hashMatch := certificate.PdfHash == pdfHash
	if !hashMatch {
		return &VerifyResult{
			IsValid:      false,
			PdfHashMatch: false,
			Status:       "pdf_tampered",
			Message:      "⚠️ FILE PDF ĐÃ BỊ THAY ĐỔI hoặc không phải file gốc được phát hành. Hash không khớp với blockchain.",
			Certificate:  certificate,
		}, nil
	}

	// --- Bước 3: Tất cả hợp lệ ---
	return &VerifyResult{
		IsValid:      true,
		PdfHashMatch: true,
		Status:       "valid",
		Message:      "✅ Văn bằng hợp lệ. File PDF này chính xác là file gốc được phát hành.",
		Certificate:  certificate,
	}, nil
}

// RevokeCertificate - Thu hồi văn bằng kèm lý do
// Cả Org1MSP (Trường ĐH) và Org2MSP (Bộ GD / Super Admin) đều được phép thu hồi
func (c *CertificateContract) RevokeCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
	reason string,
) error {
	// --- Kiểm tra quyền ---
	// Org1MSP (Trường ĐH):   University Admin thu hồi bằng trường mình
	// Org2MSP (Bộ GD):       Super Admin thu hồi bất kỳ bằng nào
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("không thể lấy MSP ID: %v", err)
	}
	if clientMSPID != "Org1MSP" && clientMSPID != "Org2MSP" {
		return fmt.Errorf("không có quyền thu hồi: cần Org1MSP (Khối Trường) hoặc Org2MSP (Bộ GD), bạn là: %s", clientMSPID)
	}

	// --- Lấy văn bằng ---
	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return err
	}

	// --- Kiểm tra trạng thái ---
	if certificate.Status == "revoked" {
		return fmt.Errorf("văn bằng %s đã bị thu hồi trước đó (lý do: %s)", certID, certificate.RevokeReason)
	}

	// --- Lý do thu hồi bắt buộc ---
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("phải cung cấp lý do thu hồi")
	}

	// --- Cập nhật trạng thái ---
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("không thể lấy timestamp: %v", err)
	}
	timestamp := time.Unix(txTimestamp.Seconds, 0)

	certificate.Status = "revoked"
	certificate.RevokeReason = reason
	certificate.UpdatedAt = timestamp

	// --- Lưu lại ---
	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		return fmt.Errorf("không thể serialize dữ liệu: %v", err)
	}

	return ctx.GetStub().PutState(certID, certificateJSON)
}

// GetCertificateHistory - Lấy toàn bộ lịch sử thay đổi của một văn bằng (audit trail)
func (c *CertificateContract) GetCertificateHistory(
	ctx contractapi.TransactionContextInterface,
	certID string,
) ([]HistoryQueryResult, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(certID)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []HistoryQueryResult
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var certificate Certificate
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &certificate)
			if err != nil {
				return nil, err
			}
		}

		record := HistoryQueryResult{
			TxID:      response.TxId,
			Timestamp: time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)),
			IsDelete:  response.IsDelete,
			Value:     certificate,
		}
		records = append(records, record)
	}

	return records, nil
}

// ============================================================
// HÀM HỖ TRỢ CHO UPDATE HISTORY (AUDIT TRAIL)
// ============================================================

// GetCertificateUpdateHistory - Lấy toàn bộ lịch sử cập nhật PDF hash của một văn bằng
// Trả về danh sách các lần cập nhật với đầy đủ thông tin: ai, khi nào, hash cũ, hash mới, lý do
func (c *CertificateContract) GetCertificateUpdateHistory(
	ctx contractapi.TransactionContextInterface,
	certID string,
) ([]UpdateRecord, error) {
	// Kiểm tra certificate có tồn tại không
	exists, err := c.CertificateExists(ctx, certID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("văn bằng %s không tồn tại", certID)
	}

	historyKey := "cert:UPDATES:" + certID
	historyJSON, err := ctx.GetStub().GetState(historyKey)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc lịch sử cập nhật: %v", err)
	}
	if historyJSON == nil {
		// Không có lịch sử cập nhật nào
		return []UpdateRecord{}, nil
	}

	var records []UpdateRecord
	err = json.Unmarshal(historyJSON, &records)
	if err != nil {
		return nil, fmt.Errorf("không thể deserialize lịch sử cập nhật: %v", err)
	}

	return records, nil
}

// ============================================================
// MAIN
// ============================================================

func main() {
	chaincode, err := contractapi.NewChaincode(&CertificateContract{})
	if err != nil {
		fmt.Printf("Lỗi khởi tạo chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Lỗi khởi động chaincode: %v\n", err)
	}
}
