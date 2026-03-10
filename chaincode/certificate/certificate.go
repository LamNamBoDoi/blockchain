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
//	Org1MSP = Khối Trường Đại Học
//	          → Backend Service dùng identity này để cấp bằng thay mặt cho từng trường
//	          → Phân quyền chi tiết (trường nào cấp bằng của trường đó) được kiểm soát ở tầng Backend/JWT
//	Org2MSP = Bộ Giáo Dục & Đào Tạo
//	          → Cơ quan giám sát độc lập, có quyền thu hồi văn bằng khi phát hiện sai phạm
//	          → Không tham gia vào quá trình cấp bằng
type CertificateContract struct {
	contractapi.Contract
}

// Certificate - Cấu trúc dữ liệu văn bằng
type Certificate struct {
	CertID         string `json:"certId"`         // Mã văn bằng (VD: CERT-BK-2025-001)
	UniversityID   string `json:"universityId"`   // Mã trường (VD: BACHKHOA, KHTN, UEH)
	UniversityName string `json:"universityName"` // Tên trường đầy đủ
	DegreeType     string `json:"degreeType"`     // Loại bằng: "Cử nhân", "Thạc sĩ", "Tiến sĩ"
	Major          string `json:"major"`          // Chuyên ngành
	StudentID      string `json:"studentId"`      // Mã sinh viên
	StudentName    string `json:"studentName"`    // Tên sinh viên
	Grade          string `json:"grade"`          // Xếp loại: "Xuất sắc", "Giỏi", "Khá", "Trung bình"
	GraduationYear string `json:"graduationYear"` // Năm tốt nghiệp
	IssueDate      string `json:"issueDate"`      // Ngày cấp (VD: "2025-06-15")
	Issuer         string `json:"issuer"`         // Người ký (VD: Hiệu trưởng)
	// ── QUAN TRỌNG NHẤT: Hash của file PDF văn bằng gốc ──────────────────────
	// Đây là trái tim của hệ thống xác minh.
	// Khi ai đó nộp file PDF, hệ thống tính SHA-256 của file đó
	// rồi so sánh với giá trị này. Nếu khớp → file gốc, không khớp → file giả/bị sửa.
	// Format: "sha256:<hex_string>" (VD: "sha256:a3f9c2d1e8b4...")
	PdfHash      string    `json:"pdfHash"`      // SHA-256 hash của file PDF gốc
	Status       string    `json:"status"`       // Trạng thái: "valid" | "revoked"
	RevokeReason string    `json:"revokeReason"` // Lý do thu hồi (nếu có)
	CreatedAt    time.Time `json:"createdAt"`    // Thời gian tạo trên blockchain
	UpdatedAt    time.Time `json:"updatedAt"`    // Thời gian cập nhật gần nhất
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
	// --- Kiểm tra quyền: Chỉ Org1MSP (Khối Trường Đại Học) được phép cấp bằng ---
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("không thể lấy MSP ID người gọi: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("không có quyền cấp văn bằng: chỉ Org1MSP (Khối Trường) được phép, bạn là: %s", clientMSPID)
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
		CertID:         certID,
		UniversityID:   universityID,
		UniversityName: universityName,
		DegreeType:     degreeType,
		Major:          major,
		StudentID:      studentID,
		StudentName:    studentName,
		Grade:          grade,
		GraduationYear: graduationYear,
		IssueDate:      issueDate,
		Issuer:         issuer,
		PdfHash:        pdfHash, // ← Khóa chống làm giả: hash của PDF gốc
		Status:         "valid",
		RevokeReason:   "",
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
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

// UpdatePdfHash - Cập nhật hash PDF khi trường phát hành lại bản PDF mới (VD: sửa lỗi in)
// Chỉ Org1MSP được phép cập nhật
func (c *CertificateContract) UpdatePdfHash(
	ctx contractapi.TransactionContextInterface,
	certID string,
	newPdfHash string,
	updateReason string,
) error {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("không thể lấy MSP ID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("chỉ Org1MSP được phép cập nhật PDF hash")
	}

	if strings.TrimSpace(newPdfHash) == "" || strings.TrimSpace(updateReason) == "" {
		return fmt.Errorf("newPdfHash và updateReason không được để trống")
	}
	if !strings.HasPrefix(newPdfHash, "sha256:") || len(newPdfHash) != 71 {
		return fmt.Errorf("newPdfHash không đúng định dạng 'sha256:<64_hex_chars>'")
	}

	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return err
	}
	if certificate.Status == "revoked" {
		return fmt.Errorf("không thể cập nhật PDF hash cho văn bằng đã bị thu hồi")
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}

	certificate.PdfHash = newPdfHash
	certificate.UpdatedAt = time.Unix(txTimestamp.Seconds, 0)

	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(certID, certificateJSON)
}

// RevokeCertificate - Thu hồi văn bằng kèm lý do
// Org1MSP (Bộ GD) hoặc Org2MSP (Cục Quản lý) đều được phép thu hồi
func (c *CertificateContract) RevokeCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
	reason string,
) error {
	// --- Kiểm tra quyền ---
	// Org1MSP (Khối Trường): thu hồi khi phát hiện sai sót nội bộ
	// Org2MSP (Bộ GD&ĐT):   thu hồi khi cơ quan nhà nước phát hiện vi phạm
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

// ============================================================
// TRUY VẤN DANH SÁCH (Yêu cầu CouchDB)
// ============================================================

// GetAllCertificates - Lấy tất cả văn bằng (dùng cho admin tổng)
func (c *CertificateContract) GetAllCertificates(
	ctx contractapi.TransactionContextInterface,
) ([]*Certificate, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var certificates []*Certificate
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var certificate Certificate
		err = json.Unmarshal(queryResponse.Value, &certificate)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, &certificate)
	}

	return certificates, nil
}

// GetCertificatesByUniversity - Lấy tất cả văn bằng của một trường đại học
// Dùng cho dashboard Admin Trường
func (c *CertificateContract) GetCertificatesByUniversity(
	ctx contractapi.TransactionContextInterface,
	universityID string,
) ([]*Certificate, error) {
	if strings.TrimSpace(universityID) == "" {
		return nil, fmt.Errorf("universityId không được để trống")
	}
	queryString := fmt.Sprintf(`{"selector":{"universityId":"%s"}}`, universityID)
	return c.getQueryResultForQueryString(ctx, queryString)
}

// GetCertificatesByStudent - Lấy tất cả văn bằng của một sinh viên
func (c *CertificateContract) GetCertificatesByStudent(
	ctx contractapi.TransactionContextInterface,
	studentID string,
) ([]*Certificate, error) {
	if strings.TrimSpace(studentID) == "" {
		return nil, fmt.Errorf("studentId không được để trống")
	}
	queryString := fmt.Sprintf(`{"selector":{"studentId":"%s"}}`, studentID)
	return c.getQueryResultForQueryString(ctx, queryString)
}

// GetCertificatesByFilter - Lọc văn bằng theo nhiều điều kiện (cho dashboard thống kê)
// Truyền chuỗi rỗng "" để bỏ qua tiêu chí đó
func (c *CertificateContract) GetCertificatesByFilter(
	ctx contractapi.TransactionContextInterface,
	universityID string,
	degreeType string,
	graduationYear string,
	status string,
) ([]*Certificate, error) {
	// Xây dựng selector động
	selector := map[string]interface{}{}

	if strings.TrimSpace(universityID) != "" {
		selector["universityId"] = universityID
	}
	if strings.TrimSpace(degreeType) != "" {
		selector["degreeType"] = degreeType
	}
	if strings.TrimSpace(graduationYear) != "" {
		selector["graduationYear"] = graduationYear
	}
	if strings.TrimSpace(status) != "" {
		selector["status"] = status
	}

	query := map[string]interface{}{"selector": selector}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("không thể tạo query: %v", err)
	}

	return c.getQueryResultForQueryString(ctx, string(queryBytes))
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
// HÀM HỖ TRỢ NỘI BỘ
// ============================================================

// getQueryResultForQueryString - Thực thi CouchDB rich query và trả về danh sách Certificate
func (c *CertificateContract) getQueryResultForQueryString(
	ctx contractapi.TransactionContextInterface,
	queryString string,
) ([]*Certificate, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var certificates []*Certificate
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var certificate Certificate
		err = json.Unmarshal(queryResponse.Value, &certificate)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, &certificate)
	}

	return certificates, nil
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
