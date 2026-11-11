package models

type SearchResult struct {
	ReceiptURL string `json:"pdf_url"`
	EventID    int64  `json:"event_id"`
	Status     string `json:"status"`
	Active     bool   `json:"active"`
	Source     string `json:"source"`
	Operation  string `json:"operation"`
	Kind       string `json:"kind"`
	Place      string `json:"place"`
	Name       string `json:"name"`
	Plate      string `json:"plate"`
	Car        string `json:"car"`
	Date       string `json:"date"`
}
