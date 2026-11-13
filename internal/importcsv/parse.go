package importcsv

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

const MaxBytes = 10 * 1024 * 1024
const MaxRows = 10000

type Row struct {
	Line        int                  `json:"line"`
	Date        string               `json:"date"`
	Operation   string               `json:"operation"`
	Name        string               `json:"name"`
	Plate       string               `json:"plate"`
	Place       string               `json:"place"`
	Car         string               `json:"car"`
	Details     models.SearchDetails `json:"details"`
	Fields      []models.CSVField    `json:"fields"`
	Values      []string             `json:"values"`
	Warnings    []string             `json:"warnings"`
	Fingerprint string               `json:"fingerprint"`
}
type Document struct {
	Hash, Encoding, Delimiter string
	Headers                   []string
	Rows                      []Row
	Raw                       []byte
	EmptyRows                 int
}

func canonicalHeader(value string) string {
	value = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))
	return strings.NewReplacer(" ", "", "_", "", "-", "").Replace(value)
}
func Parse(raw []byte) (*Document, error) {
	if len(raw) == 0 || len(raw) > MaxBytes {
		return nil, errors.New("CSV muss 1 Byte bis 10 MiB groß sein.")
	}
	text, encoding, err := decode(raw)
	if err != nil {
		return nil, err
	}
	var reader *csv.Reader
	var headers []string
	var delimiter rune
	required := []string{"DATUM", "TÄTIGKEIT", "NAME", "KENNZEICHEN", "LAGERPLATZ"}
	for _, candidate := range []rune{';', ',', '\t'} {
		trial := csv.NewReader(strings.NewReader(text))
		trial.Comma = candidate
		trial.FieldsPerRecord = -1
		head, e := trial.Read()
		if e != nil {
			continue
		}
		found := map[string]bool{}
		for _, key := range head {
			found[canonicalHeader(key)] = true
		}
		valid := true
		for _, key := range required {
			valid = valid && found[key]
		}
		if valid {
			reader = trial
			headers = head
			delimiter = candidate
			break
		}
	}
	if reader == nil {
		return nil, errors.New("CSV-Kopfzeile benötigt DATUM, TÄTIGKEIT, NAME, KENNZEICHEN und LAGERPLATZ. Trennzeichen: Semikolon, Komma oder Tab.")
	}
	seen := map[string]bool{}
	for _, header := range headers {
		key := canonicalHeader(header)
		if key == "" || seen[key] {
			return nil, errors.New("Leere oder doppelte CSV-Spaltennamen sind nicht eindeutig zuordenbar.")
		}
		seen[key] = true
	}
	doc := &Document{Hash: fmt.Sprintf("%x", sha256.Sum256(raw)), Encoding: encoding, Delimiter: string(delimiter), Headers: headers, Raw: append([]byte(nil), raw...)}
	for {
		values, e := reader.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, fmt.Errorf("CSV-Struktur fehlerhaft: %v", e)
		}
		line, _ := reader.FieldPos(0)
		empty := true
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				empty = false
			}
			if len(value) > 256*1024 {
				return nil, fmt.Errorf("Zeile %d: einzelner Wert überschreitet 256 KiB.", line)
			}
		}
		if empty {
			doc.EmptyRows++
			continue
		}
		if len(doc.Rows) >= MaxRows {
			return nil, errors.New("CSV darf höchstens 10000 nichtleere Datensätze enthalten.")
		}
		row := parseRow(headers, values, line)
		doc.Rows = append(doc.Rows, row)
	}
	if len(doc.Rows) == 0 {
		return nil, errors.New("CSV enthält keine importierbaren Vorgänge.")
	}
	return doc, nil
}
func parseRow(headers, values []string, line int) Row {
	row := Row{Line: line, Values: append([]string(nil), values...), Warnings: []string{}, Fields: []models.CSVField{}}
	mapped := map[string]string{}
	fingerprint := map[string]string{}
	for i := 0; i < len(headers) || i < len(values); i++ {
		header := ""
		if i < len(headers) {
			header = headers[i]
		} else {
			header = fmt.Sprintf("ZUSATZSPALTE_%d", i-len(headers)+1)
		}
		value := ""
		missing := i >= len(values)
		if !missing {
			value = values[i]
		}
		row.Fields = append(row.Fields, models.CSVField{Column: header, Value: value, Missing: missing})
		mapped[canonicalHeader(header)] = strings.TrimSpace(value)
		fingerprint[canonicalHeader(header)] = value
	}
	if len(values) < len(headers) {
		row.Warnings = append(row.Warnings, "Verkürzte Zeile: fehlende Spalten bleiben als nicht erfasst gekennzeichnet.")
	}
	if len(values) > len(headers) {
		row.Warnings = append(row.Warnings, "Zusätzliche Spaltenwerte werden unverändert als Zusatzspalten erhalten.")
	}
	row.Name = mapped["NAME"]
	row.Plate = mapped["KENNZEICHEN"]
	row.Place = strings.ToUpper(mapped["LAGERPLATZ"])
	row.Car = mapped["AUTO"]
	row.Operation = map[string]string{"NEUEINLAGERUNG": "storage", "EINLAGERUNG": "storage", "RADWECHSEL": "wheel_change", "AUSLAGERUNG": "release", "RESERVIERUNG": "reservation"}[strings.ToUpper(mapped["TÄTIGKEIT"])]
	if row.Operation == "" {
		row.Operation = "imported"
		row.Warnings = append(row.Warnings, "Tätigkeit wird als sonstiger Altvorgang mit Originalbezeichnung übernommen.")
	}
	location, _ := time.LoadLocation("Europe/Berlin")
	for _, layout := range []string{time.RFC3339Nano, "02.01.2006 15:04:05", "02.01.2006 15:04", "02.01.2006", "2006-01-02 15:04:05", "2006-01-02"} {
		if stamp, err := time.ParseInLocation(layout, mapped["DATUM"], location); err == nil {
			row.Date = stamp.UTC().Format(time.RFC3339Nano)
			break
		}
	}
	if row.Date == "" {
		row.Warnings = append(row.Warnings, "Datum nicht eindeutig lesbar; Original bleibt erhalten, Zeitpunkt wird als unbekannt angezeigt.")
	}
	kind := strings.ToLower(mapped["REIFENART"])
	row.Details.TireType = map[string]string{"sommer": "Sommer", "sommerreifen": "Sommer", "winter": "Winter", "winterreifen": "Winter", "allwetter": "Ganzjahr", "ganzjahr": "Ganzjahr", "ganzjahresreifen": "Ganzjahr", "allseason": "Ganzjahr", "all season": "Ganzjahr"}[kind]
	if row.Details.TireType == "" {
		row.Details.TireType = mapped["REIFENART"]
	}
	row.Details.RimType = mapped["FELGENART"]
	row.Details.Notes = mapped["HINWEISE"]
	row.Details.FrontDepth = depth(mapped["PROFILTIEFEVORNE"], &row.Warnings, "VA")
	row.Details.RearDepth = depth(mapped["PROFILTIEFEHINTEN"], &row.Warnings, "HA")
	row.Details.Accessories = map[string]models.Accessory{}
	for key, column := range map[string]string{"bolts": "RADSCHRAUBEN", "lock": "FELGENSCHLOSS", "hubcaps": "RADKAPPEN"} {
		value := mapped[column]
		if value == "" {
			continue
		}
		lower := strings.ToLower(value)
		if lower == "nein" || lower == "nicht vorhanden" || lower == "keine" || lower == "ohne" {
			row.Details.Accessories[key] = models.Accessory{}
			continue
		}
		if lower == "ja" || lower == "vorhanden" {
			row.Details.Accessories[key] = models.Accessory{Present: true}
			continue
		}
		if strings.Contains(lower, "beifahrer") || strings.Contains(lower, "radsatz") || strings.Contains(lower, "fahrzeug") || strings.Contains(lower, "beutel") || strings.Contains(lower, "kunde") {
			row.Details.Accessories[key] = models.Accessory{Present: true, Location: value}
			continue
		}
		row.Warnings = append(row.Warnings, column+": Bedeutung nicht sicher ableitbar; Originalangabe bleibt erhalten.")
	}
	serialized, _ := json.Marshal(fingerprint)
	row.Fingerprint = fmt.Sprintf("%x", sha256.Sum256(serialized))
	return row
}
func depth(raw string, warnings *[]string, axle string) string {
	if raw == "" {
		return ""
	}
	value := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(raw), "mm"))
	value = strings.ReplaceAll(value, ",", ".")
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || number < 0 || number > 20 || strings.EqualFold(value, "nan") {
		*warnings = append(*warnings, "Profiltiefe "+axle+" nicht eindeutig lesbar; Originalangabe bleibt erhalten.")
		return ""
	}
	return strconv.FormatFloat(number, 'f', -1, 64)
}
func decode(raw []byte) (string, string, error) {
	if bytes.HasPrefix(raw, []byte{0xff, 0xfe}) || bytes.HasPrefix(raw, []byte{0xfe, 0xff}) {
		if len(raw)%2 != 0 {
			return "", "", errors.New("UTF-16-Datei enthält unvollständige Zeichen.")
		}
		little := raw[0] == 0xff
		units := make([]uint16, 0, (len(raw)-2)/2)
		for i := 2; i < len(raw); i += 2 {
			var n uint16
			if little {
				n = binary.LittleEndian.Uint16(raw[i : i+2])
			} else {
				n = binary.BigEndian.Uint16(raw[i : i+2])
			}
			units = append(units, n)
		}
		encoding := "UTF-16BE"
		if little {
			encoding = "UTF-16LE"
		}
		return string(utf16.Decode(units)), encoding, nil
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	if utf8.Valid(raw) {
		return string(raw), "UTF-8", nil
	}
	// Windows-1252 is common in older workshop exports; all original bytes are retained.
	high := []rune{0x20ac, 0x0081, 0x201a, 0x0192, 0x201e, 0x2026, 0x2020, 0x2021, 0x02c6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008d, 0x017d, 0x008f, 0x0090, 0x2018, 0x2019, 0x201c, 0x201d, 0x2022, 0x2013, 0x2014, 0x02dc, 0x2122, 0x0161, 0x203a, 0x0153, 0x009d, 0x017e, 0x0178}
	var text strings.Builder
	for _, b := range raw {
		if b >= 0x80 && b <= 0x9f {
			text.WriteRune(high[int(b)-0x80])
		} else {
			text.WriteRune(rune(b))
		}
	}
	return text.String(), "Windows-1252", nil
}
