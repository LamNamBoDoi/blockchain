package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CertificateContract - Hợp đồng thông minh quản lý chứng chỉ
type CertificateContract struct {
	contractapi.Contract
}

// Certificate - Cấu trúc dữ liệu chứng chỉ
type Certificate struct {
	CertID      string    `json:"certId"`      // Mã chứng chỉ
	StudentID   string    `json:"studentId"`   // Mã sinh viên
	StudentName string    `json:"studentName"` // Tên sinh viên
	CourseName  string    `json:"courseName"`  // Tên khóa học
	Grade       string    `json:"grade"`       // Điểm số
	IssueDate   string    `json:"issueDate"`   // Ngày cấp
	Issuer      string    `json:"issuer"`      // Người cấp
	Status      string    `json:"status"`      // Trạng thái: valid (hợp lệ), revoked (đã thu hồi)
	CreatedAt   time.Time `json:"createdAt"`   // Thời gian tạo
	UpdatedAt   time.Time `json:"updatedAt"`   // Thời gian cập nhật
}

// HistoryQueryResult - Cấu trúc kết quả truy vấn lịch sử
type HistoryQueryResult struct {
	TxID      string      `json:"txId"`      // ID giao dịch
	Timestamp time.Time   `json:"timestamp"` // Thời gian
	IsDelete  bool        `json:"isDelete"`  // Đã xóa hay chưa
	Value     Certificate `json:"value"`     // Giá trị chứng chỉ
}

// InitLedger - Khởi tạo sổ cái với dữ liệu mẫu
func (c *CertificateContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("Chaincode chứng chỉ đã được khởi tạo")
	return nil
}

// CreateCertificate - Tạo chứng chỉ mới
func (c *CertificateContract) CreateCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
	studentID string,
	studentName string,
	courseName string,
	grade string,
	issueDate string,
	issuer string,
) error {
	// Kiểm tra quyền: Chỉ Org1MSP được phép tạo (VD: Giảng viên/Admin)
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("không thể lấy MSP ID người gọi: %v", err)
	}
	if clientMSPID != "Org1MSP" {
		return fmt.Errorf("bạn không có quyền tạo chứng chỉ (chỉ Org1MSP được phép, bạn là: %s)", clientMSPID)
	}
	// Kiểm tra xem chứng chỉ đã tồn tại chưa
	exists, err := c.CertificateExists(ctx, certID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("chứng chỉ %s đã tồn tại", certID)
	}

	// Lấy thời gian từ transaction
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	// FIX: Set nanoseconds to 0 to ensure determinism across peers
	timestamp := time.Unix(txTimestamp.Seconds, 0)

	// Tạo đối tượng chứng chỉ
	certificate := Certificate{
		CertID:      certID,
		StudentID:   studentID,
		StudentName: studentName,
		CourseName:  courseName,
		Grade:       grade,
		IssueDate:   issueDate,
		Issuer:      issuer,
		Status:      "valid",
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
	}

	// Chuyển đổi sang JSON
	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		return err
	}

	// Lưu vào sổ cái
	return ctx.GetStub().PutState(certID, certificateJSON)
}

// GetCertificate - Lấy thông tin chứng chỉ theo ID
func (c *CertificateContract) GetCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (*Certificate, error) {
	certificateJSON, err := ctx.GetStub().GetState(certID)
	if err != nil {
		return nil, fmt.Errorf("không thể đọc chứng chỉ: %v", err)
	}
	if certificateJSON == nil {
		return nil, fmt.Errorf("chứng chỉ %s không tồn tại", certID)
	}

	var certificate Certificate
	err = json.Unmarshal(certificateJSON, &certificate)
	if err != nil {
		return nil, err
	}

	return &certificate, nil
}

// VerifyCertificate - Xác minh chứng chỉ có hợp lệ không
func (c *CertificateContract) VerifyCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (bool, error) {
	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return false, err
	}

	return certificate.Status == "valid", nil
}

// RevokeCertificate - Thu hồi chứng chỉ
func (c *CertificateContract) RevokeCertificate(
	ctx contractapi.TransactionContextInterface,
	certID string,
) error {
	// Kiểm tra quyền: Org1MSP (Trường) hoặc Org2MSP (Bộ GD) đều được phép thu hồi
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("không thể lấy MSP ID người gọi: %v", err)
	}
	if clientMSPID != "Org1MSP" && clientMSPID != "Org2MSP" {
		return fmt.Errorf("bạn không có quyền thu hồi chứng chỉ (chỉ Org1MSP hoặc Org2MSP được phép, bạn là: %s)", clientMSPID)
	}

	certificate, err := c.GetCertificate(ctx, certID)
	if err != nil {
		return err
	}

	if certificate.Status == "revoked" {
		return fmt.Errorf("chứng chỉ %s đã bị thu hồi trước đó", certID)
	}

	// Lấy thời gian từ transaction
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	// FIX: Set nanoseconds to 0 to ensure determinism across peers
	timestamp := time.Unix(txTimestamp.Seconds, 0)

	certificate.Status = "revoked"
	certificate.UpdatedAt = timestamp

	certificateJSON, err := json.Marshal(certificate)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(certID, certificateJSON)
}

// CertificateExists - Kiểm tra chứng chỉ có tồn tại không
func (c *CertificateContract) CertificateExists(
	ctx contractapi.TransactionContextInterface,
	certID string,
) (bool, error) {
	certificateJSON, err := ctx.GetStub().GetState(certID)
	if err != nil {
		return false, fmt.Errorf("không thể đọc từ world state: %v", err)
	}

	return certificateJSON != nil, nil
}

// GetAllCertificates - Lấy tất cả chứng chỉ
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

// GetCertificateHistory - Lấy lịch sử của chứng chỉ
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

// GetCertificatesByStudent - Lấy tất cả chứng chỉ của một sinh viên
func (c *CertificateContract) GetCertificatesByStudent(
	ctx contractapi.TransactionContextInterface,
	studentID string,
) ([]*Certificate, error) {
	queryString := fmt.Sprintf(`{"selector":{"studentId":"%s"}}`, studentID)
	return c.getQueryResultForQueryString(ctx, queryString)
}

// getQueryResultForQueryString - Hàm hỗ trợ cho các truy vấn phức tạp
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

func main() {
	chaincode, err := contractapi.NewChaincode(&CertificateContract{})
	if err != nil {
		fmt.Printf("Error creating certificate chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting certificate chaincode: %v\n", err)
	}
}
