package models

type CSVField struct {
	Column  string `json:"column"`
	Value   string `json:"value"`
	Missing bool   `json:"missing,omitempty"`
}
type CSVOrigin struct {
	Filename   string     `json:"filename"`
	Line       int        `json:"line"`
	ImportedAt string     `json:"imported_at"`
	Operation  string     `json:"operation"`
	Fields     []CSVField `json:"fields"`
	Warnings   []string   `json:"warnings"`
}

// SearchDetails contains the immutable data captured for this particular event.
type SearchDetails struct {
	Import      *CSVOrigin           `json:"import,omitempty"`
	Brand       string               `json:"brand"`
	Model       string               `json:"model"`
	TireType    string               `json:"tire_type"`
	RimType     string               `json:"rim_type"`
	Front       TireSize             `json:"front"`
	Rear        TireSize             `json:"rear"`
	FrontDepth  string               `json:"front_depth"`
	RearDepth   string               `json:"rear_depth"`
	Accessories map[string]Accessory `json:"accessories"`
	Mechanic    string               `json:"mechanic"`
	Notes       string               `json:"notes"`
}

type SearchResult struct {
	Details       SearchDetails  `json:"details"`
	TreadWarnings []TreadWarning `json:"tread_warnings"`

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
