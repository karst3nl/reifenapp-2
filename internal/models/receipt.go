package models

import (
	"encoding/json"
	"fmt"
)

type Receipt struct {
	EventID   int64
	ClaimID   int64
	Operation string
	Date      string
	Place     string
	Name      string
	Plate     string
	Car       string
	Source    string
	Payload   json.RawMessage
}

func ReceiptPath(claimID int64, operation string) string {
	return fmt.Sprintf("/api/receipts/%d/%s.pdf", claimID, operation)
}
