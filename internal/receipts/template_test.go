package receipts

import (
	"bytes"
	"strings"
	"testing"
)

func TestDefaultTemplatesAreSingleA4(t *testing.T) {
	for _, operation := range []string{"storage", "wheel_change", "reservation", "release"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			template := DefaultTemplate(operation)
			if err := template.Validate(); err != nil {
				t.Fatal(err)
			}
			data, err := GenerateTemplate(ExampleReceipt(operation), Header{Company: "Werkstatt Fiktiv"}, template)
			if err != nil || !bytes.Contains(data, []byte("/Count 1")) {
				t.Fatal(err)
			}
		})
	}
}
func TestTemplateValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Template)
	}{
		{"outside", func(t *Template) { t.Elements[0].X = 209 }},
		{"negative", func(t *Template) { t.Elements[0].Y = -1 }},
		{"size", func(t *Template) { t.Elements[0].FontSize = 0 }},
		{"field", func(t *Template) { t.Elements[0].Field = "secret" }},
		{"color", func(t *Template) { t.Elements[0].Color = "url(http://example.org)" }},
		{"duplicate", func(t *Template) { t.Elements[1].ID = t.Elements[0].ID }},
		{"empty", func(t *Template) { t.Elements = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			template := DefaultTemplate("storage")
			tc.change(&template)
			if template.Validate() == nil {
				t.Fatal("invalid template accepted")
			}
		})
	}
}
func TestTemplateOverflowUsesLosslessFallback(t *testing.T) {
	template := Template{Version: 1, Elements: []Element{{ID: "name", Type: "field", Field: "name", Width: 40, Height: 5, FontSize: 12, Align: "L", Color: "#000000", Background: "#ffffff"}}}
	receipt := ExampleReceipt("storage")
	receipt.Name = strings.Repeat("W", 200)
	if _, err := GenerateTemplate(receipt, Header{}, template); err == nil {
		t.Fatal("overflow accepted")
	}
	data, fallback, err := GenerateWithFallback(receipt, Header{}, template)
	if err != nil || !fallback || !bytes.Contains(data, []byte("/Count 1")) {
		t.Fatal(fallback, err)
	}
}

func TestRGBAndShapes(t *testing.T) {
	r, g, b := rgb("#55d6ad")
	if r != 85 || g != 214 || b != 173 {
		t.Fatal(r, g, b)
	}
	t1 := Template{Version: 1, Elements: []Element{{ID: "box", Type: "box", X: 10, Y: 10, Width: 190, Height: 270, Color: "#55d6ad", Background: "#ffffff", Stroke: 0.5}, {ID: "line", Type: "line", X: 20, Y: 20, Width: 170, Height: 1, Color: "#000000", Background: "#ffffff", Stroke: 0.2}}}
	data, err := GenerateTemplate(ExampleReceipt("storage"), Header{}, t1)
	if err != nil || !bytes.Contains(data, []byte("/Count 1")) {
		t.Fatal(err)
	}
}
