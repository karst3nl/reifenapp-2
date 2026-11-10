package models

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type TireSize struct {
	Width  string `json:"width"`
	Ratio  string `json:"ratio"`
	Inches string `json:"inches"`
	Load   string `json:"load"`
	Speed  string `json:"speed"`
}
type Accessory struct {
	Present  bool   `json:"present"`
	Location string `json:"location"`
}
type StorageInput struct {
	Area       string    `json:"area"`
	Number     int       `json:"number"`
	Name       string    `json:"name"`
	Plate      string    `json:"plate"`
	Brand      string    `json:"brand"`
	Model      string    `json:"model"`
	TireType   string    `json:"tire_type"`
	RimType    string    `json:"rim_type"`
	Front      TireSize  `json:"front"`
	Rear       TireSize  `json:"rear"`
	FrontDepth string    `json:"front_depth"`
	RearDepth  string    `json:"rear_depth"`
	Hubcaps    Accessory `json:"hubcaps"`
	Bolts      Accessory `json:"bolts"`
	Lock       Accessory `json:"lock"`
	Mechanic   string    `json:"mechanic"`
	Notes      string    `json:"notes"`
}

// Initial dropdown choices, not a technical compatibility catalogue.
func StorageOptions() map[string][]string {
	ranges := func(lo, hi, step int) []string {
		var r []string
		for n := lo; n <= hi; n += step {
			r = append(r, strconv.Itoa(n))
		}
		return r
	}
	depths := []string{}
	for n := 0; n <= 40; n++ {
		depths = append(depths, strconv.FormatFloat(float64(n)/2, 'f', 1, 64))
	}
	return map[string][]string{
		"area":      {"C1", "C2", "C3", "C4", "OL", "OR"},
		"brand":     {"Audi", "BMW", "Citroën", "Dacia", "Fiat", "Ford", "Honda", "Hyundai", "Kia", "Mazda", "Mercedes-Benz", "MINI", "Nissan", "Opel", "Peugeot", "Renault", "Seat", "Škoda", "Tesla", "Toyota", "Volvo", "VW", "Sonstige"},
		"tire_type": {"Sommer", "Winter", "Ganzjahr"}, "rim_type": {"Aluminium", "Stahl", "Sonstige"},
		"width": ranges(125, 355, 10), "ratio": ranges(20, 85, 5), "inches": ranges(10, 24, 1), "load": ranges(50, 130, 1),
		"speed": {"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "B", "C", "D", "E", "F", "G", "J", "K", "L", "M", "N", "P", "Q", "R", "S", "T", "U", "H", "V", "W", "Y", "(Y)"}, "depth": depths,
	}
}
func (s StorageInput) Place() string { return fmt.Sprintf("%s.%d", s.Area, s.Number) }
func (s *StorageInput) Validate() map[string]string {
	e := map[string]string{}
	opts := StorageOptions()
	text := func(key string, p *string, required bool, max int) {
		*p = strings.TrimSpace(*p)
		if required && *p == "" {
			e[key] = "Pflichtfeld bitte ausfüllen."
		} else if utf8.RuneCountInString(*p) > max {
			e[key] = fmt.Sprintf("Höchstens %d Zeichen erlaubt.", max)
		}
	}
	choice := func(key, value, option string, required bool) {
		if value == "" && !required {
			return
		}
		for _, v := range opts[option] {
			if v == value {
				return
			}
		}
		e[key] = "Bitte einen gültigen Wert auswählen."
	}
	text("name", &s.Name, true, 200)
	text("plate", &s.Plate, true, 30)
	text("model", &s.Model, true, 100)
	text("mechanic", &s.Mechanic, false, 100)
	text("notes", &s.Notes, false, 2000)
	choice("brand", s.Brand, "brand", true)
	choice("tire_type", s.TireType, "tire_type", true)
	choice("rim_type", s.RimType, "rim_type", true)
	lo, hi := 1, 60
	switch s.Area {
	case "C1":
		hi = 200
	case "C2":
		lo = 201
		hi = 400
	case "C3", "C4", "OL", "OR":
	default:
		e["area"] = "Ungültiger Lagerbereich."
	}
	if s.Number < lo || s.Number > hi {
		e["number"] = fmt.Sprintf("Für diesen Bereich sind Nummern %d bis %d erlaubt.", lo, hi)
	}
	for _, axle := range []struct {
		key  string
		size TireSize
	}{{"front", s.Front}, {"rear", s.Rear}} {
		for key, v := range map[string]string{"width": axle.size.Width, "ratio": axle.size.Ratio, "inches": axle.size.Inches, "load": axle.size.Load, "speed": axle.size.Speed} {
			choice(axle.key+"."+key, v, key, false)
		}
	}
	choice("front_depth", s.FrontDepth, "depth", false)
	choice("rear_depth", s.RearDepth, "depth", false)
	for _, a := range []struct {
		key string
		v   *Accessory
	}{{"hubcaps", &s.Hubcaps}, {"bolts", &s.Bolts}, {"lock", &s.Lock}} {
		text(a.key+".location", &a.v.Location, a.v.Present, 200)
		if !a.v.Present {
			a.v.Location = ""
		}
	}
	return e
}
