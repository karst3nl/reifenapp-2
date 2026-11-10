package models

import "testing"

func TestStorageValidation(t *testing.T) {
	valid := func() StorageInput {
		return StorageInput{Area: "C2", Number: 201, Name: "Fiktiver Test", Plate: "MZ-AA 1", Brand: "VW", Model: "Golf", TireType: "Sommer", RimType: "Stahl", Front: TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, Rear: TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}, FrontDepth: "6.5", RearDepth: "6.0"}
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

func TestRequiredAxlesAndDepthRange(t *testing.T) {
	s := StorageInput{Area: "C1", Number: 2, Name: "Fiktiv", Plate: "MZ-AB 123", Brand: "VW", Model: "Golf", TireType: "Winter", RimType: "Stahl"}
	fields := s.Validate()
	for _, key := range []string{"front.width", "front.ratio", "front.inches", "front.load", "front.speed", "rear.width", "rear.ratio", "rear.inches", "rear.load", "rear.speed", "front_depth", "rear_depth"} {
		if fields[key] == "" {
			t.Fatalf("missing required error %s", key)
		}
	}
	s.Front = TireSize{Width: "205", Ratio: "55", Inches: "16", Load: "91", Speed: "H"}
	s.Rear = s.Front
	for _, depth := range []string{"0.0", "0.5", "9.5", "10.0"} {
		s.FrontDepth = depth
		s.RearDepth = depth
		if len(s.Validate()) != 0 {
			t.Fatal(depth, s.Validate())
		}
	}
	for _, depth := range []string{"10.5", "20.0", "-0.5", "6.2", ""} {
		s.RearDepth = depth
		if s.Validate()["rear_depth"] == "" {
			t.Fatal("invalid depth accepted", depth)
		}
	}
}
func TestPlateDistrictsAndFormat(t *testing.T) {
	for _, plate := range []string{"MZ-AB 123", "BIN-X 1", "RÜD-AB 1234H", "WI-E 123E"} {
		if !validPlate(plate) {
			t.Fatal(plate)
		}
	}
	for _, plate := range []string{"XYZ-AB 123", "MZ-ABC 123", "MZ-AB 0123", "MZ-AB 12345", "MZ", ""} {
		if validPlate(plate) {
			t.Fatal(plate)
		}
	}
	opts := StorageOptions()
	if opts["plate_district"][0] != "MZ" || opts["plate_district"][1] != "BIN" {
		t.Fatal("regional order")
	}
	seen := map[string]bool{}
	for _, d := range opts["plate_district"] {
		if seen[d] {
			t.Fatal("duplicate", d)
		}
		seen[d] = true
	}
	if len(opts["depth"]) != 21 || opts["depth"][20] != "10.0" {
		t.Fatal("depth options")
	}
}
