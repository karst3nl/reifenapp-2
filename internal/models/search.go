package models

type SearchResult struct {
	Operation string `json:"operation"`
	Kind      string `json:"kind"`
	Place     string `json:"place"`
	Name      string `json:"name"`
	Plate     string `json:"plate"`
	Car       string `json:"car"`
	Date      string `json:"date"`
}
