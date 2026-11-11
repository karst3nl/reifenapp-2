package receipts

import (
	"encoding/json"
	"fmt"
	"os"
	"unicode/utf8"
)

// Header contains locally configured workshop details, never customer data.
type Header struct {
	Company string `json:"company"`
	Street  string `json:"street"`
	City    string `json:"city"`
	Phone   string `json:"phone"`
}

func LoadHeader(path string) (Header, error) {
	header := Header{Company: "Reifenapp 2.0"}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return header, nil
	}
	if err != nil {
		return header, err
	}
	if err = json.Unmarshal(data, &header); err != nil {
		return header, fmt.Errorf("ungültige Firmenkopf-Konfiguration")
	}
	if err = header.validate(); err != nil {
		return header, err
	}
	return header, nil
}
func (h Header) validate() error {
	for _, value := range []string{h.Company, h.Street, h.City, h.Phone} {
		if utf8.RuneCountInString(value) > 100 {
			return fmt.Errorf("Firmenkopf: höchstens 100 Zeichen je Zeile erlaubt")
		}
	}
	return nil
}
