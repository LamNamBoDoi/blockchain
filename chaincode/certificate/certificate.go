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
//
// Audit trail (lưu vào Certificate struct):
//
//	CreatedBy / CreatedByMSP : ai đã tạo bằng trên blockchain
//	RevokedBy / RevokedByMSP / RevokedAt : ai đã thu hồi bằng
type CertificateContract struct {
	contractapi.Contract
}

// Certificate - Cấu trúc dữ liệu văn bằng
type Certificate struct {
	CertID           string `json:"certId"`          // Mã văn bằng (VD: CERT-BK-2025-001)
	UniversityID     string `json:"universityId"`    // Mã trường (VD: BACHKHOA, KHTN, UEH)
	UniversityName   string `json:"universityName"`  // Tên trường đầy đủ
	DegreeType       string `json:"degreeType"`      // Loại bằng: "Cử nhân", "Thạc sĩ", "Tiến sĩ"
	Major            string `json:"major"`           // Chuyên ngành
	StudentID        string `json:"studentId"`       // Mã sinh viên
	StudentName      string `json:"studentName"`     // Tên sinh viên
	Grade            string `json:"grade"`           // Xếp loại: "Xuất sắc", "Giỏi", "Khá", "Trung bình"
	GraduationYear   string `json:"graduationYear"`  // Năm tốt nghiệp
	IssueDate        string `json:"issueDate"`       // Ngày cấp (VD: "2025-06-15")
	Issuer           string `json:"issuer"`          // Người ký (VD: Hiệu trưởng)
	PdfHash          string `json:"pdfHash"`         // SHA-256 hash của file PDF gốc
	Status           string `json:"status"`          // Trạng thái: "valid" | "revoked"
	CreatedBy        string `json:"createdBy"`       // Enrollment ID của người tạo
	CreatedByMSP     string `json:"createdByMsp"`    // MSP ID của người tạo
	RevokeReason     string `json:"revokeReason"`   // Lý do thu hồi
	RevokedBy        string `json:"revokedBy"`      // Enrollment ID người thu hồi
	RevokedByMSP     string `json:"revokedByMsp"`  // MSP ID người thu hồi
	RevokedAt        string `json:"revokedAt"`      // Thời gian thu hồi (RFC3339)
	UpdateCount      int    `json:"updateCount"`     // Số lần cập nhật PDF hash
	LastUpdateReason string `json:"lastUpdateReason"` // Lý do cập nhật gần nhất
	CreatedAt        string `json:"createdAt"`      // Thời gian tạo (RFC3339)
	UpdatedAt        string `json:"updatedAt"`      // Thời gian cập nhật (RFC3339)
}

// UpdateRecord - Bản ghi lịch sử cập nhật PDF hash
type UpdateRecord struct {
	TxID       string `json:"txId"`
	Timestamp  string `json:"timestamp"`
	OldPdfHash string `json:"oldPdfHash"`
	NewPdfHash string `json:"newPdfHash"`
	Reason     string `json:"reason"`
	UpdatedBy  string `json:"updatedBy"`
}

// HistoryQueryResult - Kết quả truy vấn lịch sử giao dịch
type HistoryQueryResult struct {
	TxID      string      `json:"txId"`
	Timestamp string      `json:"timestamp"`
	IsDelete  bool        `json:"isDelete"`
	Value     Certificate `json:"value"`
}

type VerifyResult struct {
	IsValid      bool         `json:"isValid"`
	PdfHashMatch bool         `json:"pdfHashMatch"`
	Status       string       `json:"status"`
	Message      string       `json:"message"`
	Certificate  *Certificate `json:"certificate"`
}

// ============================================================
// KHỞI TẠO
// ============================================================

func (c *CertificateContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("Chaincode Van Bang da duoc khoi tao thanh cong")
	return nil
}

// ============================================================
// CRUD CƠ BẢN
// ============================================================

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
	pdfHash string,
	issuerID string,
) error {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("khong the lay MSP ID nguoi goi: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("khong co quyen cap van bang: chi Org1MSP (Truong DH) duoc phep, ban la: %s", clientMSPID)
	}

	if strings.TrimSpace(certID) == "" || strings.TrimSpace(universityID) == "" ||
		strings.TrimSpace(studentID) == "" || strings.TrimSpace(studentName) == "" {
		return fmt.Errorf("certId, universityId, studentId, studentName khong duoc de trong")
	}

	if strings.TrimSpace(pdfHash) == "" {
		return fmt.Errorf("pdfHash khong duoc de trong")
	}
	if !strings.HasPrefix(pdfHash, "sha256:") || len(pdfHash) != 71 {
		return fmt.Errorf("pdfHash khong dung dinh dang, can 'sha256:<64_hex_chars>', nhan duoc: %s", pdfHash)
	}

	exists, err := c.CertificateExists(ctx, certID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("van bang %s da ton tai tren blockchain", certID)
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("khong the lay timestamp giao dich: %v", err)
	}
	ts := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)

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
		PdfHash:          pdfHash,
		Status:           "valid",
		CreatedBy:        issuerID,
		CreatedByMSP:     clientMSPID,
		RevokeReason:     "",
		RevokedBy:        "",
		RevokedByMSP:     "",
		RevokedAt:        "",
		UpdateCount:       0,
		LastUpdateReason: "",
		CreatedAt:        ts,
		UpdatedAt:        ts,
	}

	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		return fmt.Errorf("khong the serialize du lieu: %v", err)
	}

	return ctx.GetStub().PutState(certID, certificateJSON)
}

func (c *CertificateContract) GetCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (*Certificate, error) {
	certificateJSON, err := ctx.GetStub().GetState(certID)
	if err != nil {
		return nil, fmt.Errorf("khong the doc du lieu: %v", err)
	}
	if certificateJSON == nil {
		return nil, fmt.Errorf("van bang %s khong ton tai", certID)
	}

	var certificate Certificate
	err = json.Unmarshal(certificateJSON, &certificate)
	if err != nil {
		return nil, fmt.Errorf("khong the deserialize du lieu: %v", err)
	}

	return &certificate, nil
}

func (c *CertificateContract) CertificateExists(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (bool, error) {
	certificateJSON, err := ctx.GetStub().GetState(certID)
	if err != nil {
		return false, fmt.Errorf("loi khi doc world state: %v", err)
	}
	return certificateJSON != nil, nil
}

// ============================================================
// XÁC MINH & THU HỒI
// ============================================================

func (c *CertificateContract) VerifyCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (*VerifyResult, error) {
	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return &VerifyResult{
			IsValid: false,
			Status:  "not_found",
			Message: fmt.Sprintf("Khong tim thay van bang voi ma: %s", certID),
		}, nil
	}

	if certificate.Status == "revoked" {
		revokeInfo := fmt.Sprintf("Van bang da bi thu hoi. Ly do: %s", certificate.RevokeReason)
		if certificate.RevokedBy != "" {
			revokeInfo += fmt.Sprintf(" | Nguoi thu hoi: %s (%s)", certificate.RevokedBy, certificate.RevokedByMSP)
		}
		if certificate.RevokedAt != "" {
			revokeInfo += fmt.Sprintf(" | Luc: %s", certificate.RevokedAt)
		}
		return &VerifyResult{
			IsValid:     false,
			Status:      "revoked",
			Message:     revokeInfo,
			Certificate: certificate,
		}, nil
	}

	return &VerifyResult{
		IsValid:     true,
		Status:      "valid",
		Message:     "Van bang hop le.",
		Certificate: certificate,
	}, nil
}

func (c *CertificateContract) VerifyWithPdfHash(
	ctx contractapi.TransactionContextInterface,
	certID string,
	pdfHash string,
) (*VerifyResult, error) {
	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return &VerifyResult{
			IsValid: false,
			Status:  "not_found",
			Message: fmt.Sprintf("Khong tim thay van bang voi ma: %s", certID),
		}, nil
	}

	if certificate.Status == "revoked" {
		revokeInfo := fmt.Sprintf("Van bang da bi thu hoi. Ly do: %s", certificate.RevokeReason)
		if certificate.RevokedBy != "" {
			revokeInfo += fmt.Sprintf(" | Nguoi thu hoi: %s (%s)", certificate.RevokedBy, certificate.RevokedByMSP)
		}
		if certificate.RevokedAt != "" {
			revokeInfo += fmt.Sprintf(" | Luc: %s", certificate.RevokedAt)
		}
		return &VerifyResult{
			IsValid:      false,
			PdfHashMatch: false,
			Status:       "revoked",
			Message:      revokeInfo,
			Certificate:  certificate,
		}, nil
	}

	hashMatch := certificate.PdfHash == pdfHash
	if !hashMatch {
		return &VerifyResult{
			IsValid:      false,
			PdfHashMatch: false,
			Status:       "pdf_tampered",
			Message:      "FILE PDF da bi thay doi hoac khong phai file goc duoc phat hanh. Hash khong khop voi blockchain.",
			Certificate:  certificate,
		}, nil
	}

	return &VerifyResult{
		IsValid:      true,
		PdfHashMatch: true,
		Status:       "valid",
		Message:      "Van bang hop le. File PDF nay chinh xac la file goc duoc phat hanh.",
		Certificate:  certificate,
	}, nil
}

func (c *CertificateContract) RevokeCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
	reason string,
	revokerID string,
) error {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("khong the lay MSP ID: %v", err)
	}

	if clientMSPID != "Org1MSP" && clientMSPID != "Org2MSP" {
		return fmt.Errorf("khong co quyen thu hoi: can Org1MSP hoac Org2MSP, ban la: %s", clientMSPID)
	}

	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return err
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("khong the lay timestamp: %v", err)
	}
	ts := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)

	if certificate.Status == "revoked" {
		return fmt.Errorf("van bang %s da bi thu hoi truoc do (ly do: %s)", certID, certificate.RevokeReason)
	}

	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("phai cung cap ly do thu hoi")
	}

	certificate.Status = "revoked"
	certificate.RevokeReason = reason
	certificate.RevokedBy = revokerID
	certificate.RevokedByMSP = clientMSPID
	certificate.RevokedAt = ts
	certificate.UpdatedAt = ts

	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		return fmt.Errorf("khong the serialize du lieu: %v", err)
	}

	return ctx.GetStub().PutState(certID, certificateJSON)
}

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

		ts := ""
		if response.Timestamp != nil {
			ts = time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).UTC().Format(time.RFC3339)
		}

		record := HistoryQueryResult{
			TxID:      response.TxId,
			Timestamp: ts,
			IsDelete:  response.IsDelete,
			Value:     certificate,
		}
		records = append(records, record)
	}

	return records, nil
}

func (c *CertificateContract) GetCertificateUpdateHistory(
	ctx contractapi.TransactionContextInterface,
	certID string,
) ([]UpdateRecord, error) {
	exists, err := c.CertificateExists(ctx, certID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("van bang %s khong ton tai", certID)
	}

	historyKey := "cert:UPDATES:" + certID
	historyJSON, err := ctx.GetStub().GetState(historyKey)
	if err != nil {
		return nil, fmt.Errorf("khong the doc lich su cap nhat: %v", err)
	}
	if historyJSON == nil {
		return []UpdateRecord{}, nil
	}

	var records []UpdateRecord
	err = json.Unmarshal(historyJSON, &records)
	if err != nil {
		return nil, fmt.Errorf("khong the deserialize lich su cap nhat: %v", err)
	}

	return records, nil
}

// ============================================================
// QUERY ALL & UPDATE PDF HASH
// ============================================================

func (c *CertificateContract) QueryAllCertificates(
	ctx contractapi.TransactionContextInterface,
) ([]*Certificate, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var certs []*Certificate
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(response.Key, "cert:") {
			continue
		}
		var cert Certificate
		if err := json.Unmarshal(response.Value, &cert); err != nil {
			continue
		}
		certs = append(certs, &cert)
	}
	return certs, nil
}

func (c *CertificateContract) UpdatePdfHash(
	ctx contractapi.TransactionContextInterface,
	certID string,
	newPdfHash string,
	reason string,
) error {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("khong the lay MSP ID: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("chi Org1MSP duoc phep cap nhat PDF hash")
	}

	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return err
	}
	if certificate.Status == "revoked" {
		return fmt.Errorf("khong the cap nhat van bang da bi thu hoi")
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("phai cung cap ly do cap nhat")
	}
	if !strings.HasPrefix(newPdfHash, "sha256:") || len(newPdfHash) != 71 {
		return fmt.Errorf("pdfHash khong dung dinh dang, can 'sha256:<64_hex_chars>', nhan duoc: %s", newPdfHash)
	}

	updaterID, found, err := ctx.GetClientIdentity().GetAttributeValue("hf.EnrollmentID")
	if err != nil || !found {
		updaterID, _ = ctx.GetClientIdentity().GetID()
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("khong the lay timestamp: %v", err)
	}
	ts := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)

	historyKey := "cert:UPDATES:" + certID
	var records []UpdateRecord
	existing, err := ctx.GetStub().GetState(historyKey)
	if err == nil && existing != nil {
		json.Unmarshal(existing, &records)
	}
	records = append(records, UpdateRecord{
		TxID:       ctx.GetStub().GetTxID(),
		Timestamp:  ts,
		OldPdfHash: certificate.PdfHash,
		NewPdfHash: newPdfHash,
		Reason:     reason,
		UpdatedBy:  updaterID,
	})
	historyJSON, _ := json.Marshal(records)
	ctx.GetStub().PutState(historyKey, historyJSON)

	certificate.PdfHash = newPdfHash
	certificate.UpdateCount++
	certificate.LastUpdateReason = reason
	certificate.UpdatedAt = ts

	certJSON, _ := json.Marshal(certificate)
	return ctx.GetStub().PutState(certID, certJSON)
}

// ============================================================
// MAIN
// ============================================================

func main() {
	chaincode, err := contractapi.NewChaincode(&CertificateContract{})
	if err != nil {
		fmt.Printf("Loi khoi tao chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Loi khoi dong chaincode: %v\n", err)
	}
}
