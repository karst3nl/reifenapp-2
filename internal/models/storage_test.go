package models

import "testing"

func TestStorageValidation(t *testing.T) {
	valid := func() StorageInput {
		return StorageInput{Area: "C2", Number: 201, Name: "Fiktiver Test", Plate: "TEST-A 1", Brand: "VW", Model: "Golf", TireType: "Sommer", RimType: "Stahl"}
	}
	tests := []struct {
		name, key string
		change    func(*StorageInput)
	}{
		{"missing name", "name", func(s *StorageInput) { s.Name = "  " }},
		{"missing plate", "plate", func(s *StorageInput) { s.Plate = "" }},
		{"missing model", "model", func(s *StorageInput) { s.Model = "" }},
		{"invalid area", "area", func(s *StorageInput) { s.Area = "C5" }},
		{"C2 range", "number", func(s *StorageInput) { s.Number = 200 }},
		{"brand", "brand", func(s *StorageInput) { s.Brand = "unknown" }},
		{"tire type", "tire_type", func(s *StorageInput) { s.TireType = "" }},
		{"rim type", "rim_type", func(s *StorageInput) { s.RimType = "" }},
		{"depth step", "front_depth", func(s *StorageInput) { s.FrontDepth = "6.2" }},
		{"width", "front.width", func(s *StorageInput) { s.Front.Width = "999" }},
		{"accessory location", "hubcaps.location", func(s *StorageInput) { s.Hubcaps.Present = true }},
	}
	s := valid()
	if e := s.Validate(); len(e) != 0 {
		t.Fatal(e)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := valid()
			tc.change(&s)
			if s.Validate()[tc.key] == "" {
				t.Fatal("expected field error")
			}
		})
	}
	for _, tc := range []struct {
		area   string
		lo, hi int
	}{{"C1", 1, 200}, {"C2", 201, 400}, {"C3", 1, 60}, {"C4", 1, 60}, {"OL", 1, 60}, {"OR", 1, 60}} {
		for _, n := range []int{tc.lo, tc.hi} {
			s := valid()
			s.Area = tc.area
			s.Number = n
			if len(s.Validate()) > 0 {
				t.Fatalf("boundary %s.%d", tc.area, n)
			}
		}
	}
}
