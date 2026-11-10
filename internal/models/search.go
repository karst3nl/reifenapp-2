package models

type SearchResult struct {
	Place string `json:"place"`
	Name  string `json:"name"`
	Plate string `json:"plate"`
	Car   string `json:"car"`
	Date  string `json:"date"`
}
