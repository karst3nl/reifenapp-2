package receipts

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func fictitiousLogo(t *testing.T, width, height int) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 40, G: 180, B: 120, A: uint8((x + y) % 256)})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestLogoTemplatePDFAndTransparency(t *testing.T) {
	logo := fictitiousLogo(t, 120, 40)
	for _, operation := range []string{"storage", "wheel_change", "reservation", "release"} {
		template := DefaultTemplate(operation)
		e := Element{ID: "fictitious-logo", Type: "image", Image: logo, X: 140, Y: 2, Width: 60, Height: 20, Color: "#000000", Background: "#ffffff"}
		template.Elements = append(template.Elements, e)
		data, err := GenerateTemplate(ExampleReceipt(operation), Header{}, template)
		if err != nil || !bytes.Contains(data, []byte("/Count 1")) || !bytes.Contains(data, []byte("/Subtype /Image")) || !bytes.Contains(data, []byte("/SMask")) {
			t.Fatalf("%s: logo/transparency missing: %v", operation, err)
		}
	}
}
func TestLogoValidationRejectsUnsafeAndMalformedData(t *testing.T) {
	good := fictitiousLogo(t, 10, 10)
	for _, value := range []string{"", "https://example.invalid/logo.png", "/tmp/logo.png", "data:image/svg+xml;base64,PHN2Zy8+", "data:image/png;base64,invalid", good[:len(good)-20], "data:image/png;base64," + strings.Repeat("A", base64.StdEncoding.EncodedLen(MaxLogoBytes)+4), fictitiousLogo(t, 2049, 1)} {
		template := DefaultTemplate("storage")
		template.Elements[0].Type = "image"
		template.Elements[0].Image = value
		if err := template.Validate(); err == nil {
			t.Fatal("invalid logo accepted")
		}
	}
	template := DefaultTemplate("storage")
	template.Elements[0].Image = good
	if err := template.Validate(); err == nil {
		t.Fatal("hidden image accepted on text element")
	}
}
func TestLogoTemplateBudget(t *testing.T) {
	// Repeated noisy PNGs exercise the aggregate limit independently of the per-logo limit.
	img := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	var n uint32 = 17
	for i := range img.Pix {
		n = n*1664525 + 1013904223
		img.Pix[i] = byte(n >> 24)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	template := DefaultTemplate("reservation")
	logo := "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
	if b.Len() > MaxLogoBytes {
		t.Fatal("fixture too big")
	}
	for i := 0; i < MaxTemplateLogoBytes/b.Len()+1; i++ {
		template.Elements = append(template.Elements, Element{ID: fmt.Sprint("logo-", i), Type: "image", Image: logo, X: 1, Y: 1, Width: 10, Height: 10, Color: "#000000", Background: "#ffffff"})
	}
	if err := template.Validate(); err == nil || !strings.Contains(err.Error(), "4 MiB") {
		t.Fatal("budget not enforced", err)
	}
}
