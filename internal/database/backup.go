package database

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BackupInfo struct {
	Database      string `json:"database"`
	SchemaVersion int    `json:"schema_version"`
}
type BackupManifest struct {
	Format         string         `json:"format"`
	CreatedAt      time.Time      `json:"created_at"`
	Source         BackupInfo     `json:"source"`
	DatabaseSHA256 string         `json:"database_sha256"`
	Tables         map[string]int `json:"tables"`
}
type Backup struct {
	Path, Filename string
	Manifest       BackupManifest
	directory      string
}

func (b *Backup) Close() error { return os.RemoveAll(b.directory) }

func ReadBackupInfo(ctx context.Context, db *sql.DB) (BackupInfo, error) {
	var result BackupInfo
	rows, err := db.QueryContext(ctx, "PRAGMA database_list")
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var seq int
		var name, path string
		if err = rows.Scan(&seq, &name, &path); err != nil {
			rows.Close()
			return result, err
		}
		if name == "main" {
			result.Database = filepath.Base(path)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	err = db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&result.SchemaVersion)
	return result, err
}

// CreateBackup exports only the immutable VACUUM snapshot, never a mixture of
// live tables. The caller closes the returned temporary bundle after download.
func CreateBackup(ctx context.Context, db *sql.DB, receiptHeader []byte) (*Backup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := ReadBackupInfo(ctx, db)
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "reifenapp-backup-")
	if err != nil {
		return nil, err
	}
	result := &Backup{directory: directory, Filename: "reifenapp-backup-" + time.Now().UTC().Format("20060102T150405.000000000Z") + ".zip", Manifest: BackupManifest{Format: "reifenapp-backup-v1", CreatedAt: time.Now().UTC(), Source: info, Tables: map[string]int{}}}
	fail := func(err error) (*Backup, error) { result.Close(); return nil, err }
	snapshot := filepath.Join(directory, "datenbank.sqlite")
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", snapshot); err != nil {
		return fail(err)
	}
	if err = os.Chmod(snapshot, 0600); err != nil {
		return fail(err)
	}
	u := url.URL{Scheme: "file", Path: snapshot}
	copyDB, err := sql.Open("sqlite3", u.String()+"?mode=ro&immutable=1")
	if err != nil {
		return fail(err)
	}
	copyDB.SetMaxOpenConns(1)
	var integrity string
	err = copyDB.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity)
	if err != nil || integrity != "ok" {
		copyDB.Close()
		if err == nil {
			err = fmt.Errorf("Backup-Integritätsprüfung fehlgeschlagen")
		}
		return fail(err)
	}
	tables, err := backupTables(ctx, copyDB)
	if err == nil {
		err = exportBackup(ctx, copyDB, directory, tables, &result.Manifest)
	}
	copyDB.Close()
	if err != nil {
		return fail(err)
	}
	file, err := os.Open(snapshot)
	if err != nil {
		return fail(err)
	}
	digest := sha256.New()
	_, err = io.Copy(digest, file)
	file.Close()
	if err != nil {
		return fail(err)
	}
	result.Manifest.DatabaseSHA256 = fmt.Sprintf("%x", digest.Sum(nil))
	manifest, err := json.MarshalIndent(result.Manifest, "", "  ")
	if err != nil {
		return fail(err)
	}
	if err = os.WriteFile(filepath.Join(directory, "manifest.json"), manifest, 0600); err != nil {
		return fail(err)
	}
	if len(receiptHeader) > 0 {
		if !json.Valid(receiptHeader) {
			return fail(fmt.Errorf("Ungültiger Firmenkopf"))
		}
		if err = os.WriteFile(filepath.Join(directory, "receipt-header.json"), receiptHeader, 0600); err != nil {
			return fail(err)
		}
	}
	if err = os.WriteFile(filepath.Join(directory, "BITTE-LESEN.txt"), []byte(backupReadme), 0600); err != nil {
		return fail(err)
	}
	result.Path = filepath.Join(directory, result.Filename)
	out, err := os.OpenFile(result.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fail(err)
	}
	archive := zip.NewWriter(out)
	names := []string{"datenbank.sqlite", "uebersicht.html", "daten.json", "schema.sql", "manifest.json", "BITTE-LESEN.txt"}
	if len(receiptHeader) > 0 {
		names = append(names, "receipt-header.json")
	}
	for _, name := range names {
		if err = ctx.Err(); err != nil {
			break
		}
		var entry io.Writer
		zipHeader := zip.FileHeader{Name: strings.TrimSuffix(result.Filename, ".zip") + "/" + name, Method: zip.Deflate}
		zipHeader.SetMode(0600)
		zipHeader.SetModTime(result.Manifest.CreatedAt)
		entry, err = archive.CreateHeader(&zipHeader)
		if err != nil {
			break
		}
		var source *os.File
		source, err = os.Open(filepath.Join(directory, name))
		if err != nil {
			break
		}
		_, err = io.Copy(contextWriter{ctx, entry}, source)
		source.Close()
		if err != nil {
			break
		}
	}
	closeErr := archive.Close()
	fileErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = fileErr
	}
	if err != nil {
		return fail(err)
	}
	return result, nil
}

type backupTable struct{ Name string }

func backupTables(ctx context.Context, db *sql.DB) ([]backupTable, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []backupTable{}
	for rows.Next() {
		var table backupTable
		if err = rows.Scan(&table.Name); err != nil {
			return nil, err
		}
		result = append(result, table)
	}
	return result, rows.Err()
}
func quotedIdentifier(name string) string {
	return "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\""
}

func exportBackup(ctx context.Context, db *sql.DB, directory string, tables []backupTable, manifest *BackupManifest) (err error) {
	jsonFile, err := os.OpenFile(filepath.Join(directory, "daten.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if e := jsonFile.Close(); err == nil {
			err = e
		}
	}()
	htmlFile, err := os.OpenFile(filepath.Join(directory, "uebersicht.html"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if e := htmlFile.Close(); err == nil {
			err = e
		}
	}()
	jw, hw := bufio.NewWriter(jsonFile), bufio.NewWriter(htmlFile)
	fmt.Fprint(jw, "{\"format\":\"reifenapp-backup-v1\",\"tables\":{")
	fmt.Fprint(hw, backupHTMLStart)
	fmt.Fprintf(hw, "<p>Erstellt: %s · Datenbank: %s · Schemaversion: %d</p><p>Alle Tabellen des gesicherten Datenstands. Strukturierte Angaben sind aufklappbar. Der Admin-Passwort-Hash befindet sich ausschließlich in der vollständigen Datenbankkopie.</p><nav>", html.EscapeString(manifest.CreatedAt.Format(time.RFC3339)), html.EscapeString(manifest.Source.Database), manifest.Source.SchemaVersion)
	for _, table := range tables {
		fmt.Fprintf(hw, "<a href=\"#%s\">%s</a> ", html.EscapeString(table.Name), html.EscapeString(tableTitle(table.Name)))
	}
	fmt.Fprint(hw, "</nav>")
	encoder := json.NewEncoder(jw)
	for index, table := range tables {
		if err = ctx.Err(); err != nil {
			return err
		}
		rows, e := db.QueryContext(ctx, "SELECT * FROM "+quotedIdentifier(table.Name))
		if e != nil {
			return e
		}
		columns, e := rows.Columns()
		if e != nil {
			rows.Close()
			return e
		}
		if index > 0 {
			fmt.Fprint(jw, ",")
		}
		key, _ := json.Marshal(table.Name)
		fmt.Fprintf(jw, "%s:[", key)
		fmt.Fprintf(hw, "<section id=\"%s\"><h2>%s <small>(%s)</small></h2><div class=\"table\"><table><thead><tr>", html.EscapeString(table.Name), html.EscapeString(tableTitle(table.Name)), html.EscapeString(table.Name))
		for _, column := range columns {
			fmt.Fprintf(hw, "<th>%s<small>%s</small></th>", html.EscapeString(columnTitle(column)), html.EscapeString(column))
		}
		fmt.Fprint(hw, "</tr></thead><tbody>")
		count := 0
		for rows.Next() {
			if err = ctx.Err(); err != nil {
				rows.Close()
				return err
			}
			values := make([]any, len(columns))
			targets := make([]any, len(columns))
			for i := range values {
				targets[i] = &values[i]
			}
			if e = rows.Scan(targets...); e != nil {
				rows.Close()
				return e
			}
			record := map[string]any{}
			fmt.Fprint(hw, "<tr>")
			for i, column := range columns {
				value := values[i]
				if table.Name == "admin_account" && column == "password_hash" {
					value = "[Nur in datenbank.sqlite enthalten]"
				}
				record[column] = value
				fmt.Fprint(hw, "<td>")
				text := backupText(value)
				if str, ok := value.(string); ok && json.Valid([]byte(str)) && (strings.HasPrefix(strings.TrimSpace(str), "{") || strings.HasPrefix(strings.TrimSpace(str), "[")) {
					var pretty any
					json.Unmarshal([]byte(str), &pretty)
					formatted, _ := json.MarshalIndent(pretty, "", "  ")
					fmt.Fprintf(hw, "<details><summary>Angaben anzeigen</summary><pre>%s</pre></details>", html.EscapeString(string(formatted)))
				} else {
					fmt.Fprint(hw, html.EscapeString(text))
				}
				fmt.Fprint(hw, "</td>")
			}
			fmt.Fprint(hw, "</tr>")
			if count > 0 {
				fmt.Fprint(jw, ",")
			}
			if e = encoder.Encode(record); e != nil {
				rows.Close()
				return e
			}
			count++
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		manifest.Tables[table.Name] = count
		fmt.Fprint(jw, "]")
		fmt.Fprintf(hw, "</tbody></table></div><p>%d Datensätze</p></section>", count)
	}
	fmt.Fprint(jw, "}}")
	fmt.Fprint(hw, "</body></html>")
	if err = jw.Flush(); err != nil {
		return err
	}
	if err = hw.Flush(); err != nil {
		return err
	}
	return exportSchema(ctx, db, filepath.Join(directory, "schema.sql"))
}
func exportSchema(ctx context.Context, db *sql.DB, path string) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if e := file.Close(); err == nil {
			err = e
		}
	}()
	rows, err := db.QueryContext(ctx, "SELECT sql FROM sqlite_schema WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%' ORDER BY CASE type WHEN 'table' THEN 0 WHEN 'view' THEN 1 WHEN 'index' THEN 2 ELSE 3 END,name")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var statement string
		if err = rows.Scan(&statement); err != nil {
			return err
		}
		if _, err = fmt.Fprintln(file, statement+";"); err != nil {
			return err
		}
	}
	return rows.Err()
}
func backupText(value any) string {
	if value == nil {
		return "—"
	}
	if data, ok := value.([]byte); ok {
		return "Base64: " + base64.StdEncoding.EncodeToString(data)
	}
	return fmt.Sprint(value)
}
func tableTitle(name string) string {
	titles := map[string]string{"accessory_options": "Zubehör-Auswahllisten", "label_printer_settings": "Etikettendrucker und Vorlage", "label_print_jobs": "Etikettenaufträge", "inactivity_settings": "Frist für fehlende Lagerbewegung", "csv_import_batches": "CSV-Importdateien", "csv_import_rows": "CSV-Originalzeilen", "customers": "Kunden", "vehicles": "Fahrzeuge", "tire_sets": "Radsätze", "storage_places": "Lagerplätze", "storage_areas": "Lagerorte", "storage_claims": "Belegungen und Reservierungen", "storage_details": "Reifendaten und Zubehör", "history_events": "Vorgangshistorie", "storage_releases": "Auslagerungen", "pdf_layouts": "PDF-Vorlagen und Logos", "admin_account": "Admin-Konto", "schema_migrations": "Schemaversionen"}
	if title := titles[name]; title != "" {
		return title
	}
	return name
}

func columnTitle(name string) string {
	titles := map[string]string{"id": "Nummer", "customer_id": "Kundennummer", "vehicle_id": "Fahrzeugnummer", "tire_set_id": "Radsatznummer", "claim_id": "Belegungsnummer", "place_id": "Lagerplatz", "name": "Name", "plate": "Kennzeichen", "car": "Fahrzeug", "occurred_at": "Zeitpunkt", "operation": "Vorgangsart", "payload": "Vorgangsdaten", "source": "Datenquelle", "tire_type": "Reifenart", "rim_type": "Felgenart", "front_size": "Reifengröße VA", "rear_size": "Reifengröße HA", "front_depth": "Profiltiefe VA", "rear_depth": "Profiltiefe HA", "accessories": "Zubehör", "mechanic": "Monteur", "notes": "Hinweise", "layout": "PDF-Vorlage", "password_hash": "Passwort-Hash", "username": "Benutzername", "number": "Lagernummer", "area": "Lagerort", "code": "Kennung", "first_number": "Erste Nummer", "last_number": "Letzte Nummer", "kind": "Art", "status": "Status", "updated_at": "Aktualisiert", "released_at": "Ausgelagert am", "enabled": "Aktiv", "revision": "Revision", "must_change": "Passwortwechsel nötig", "version": "Schemaversion"}
	if title := titles[name]; title != "" {
		return title
	}
	return name
}

type contextWriter struct {
	ctx    context.Context
	writer io.Writer
}

func (w contextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

const backupHTMLStart = `<!DOCTYPE html><html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'"><title>Reifenapp · Datenbanksicherung</title><style>body{font:15px/1.6 system-ui,sans-serif;color:#20201e;background:#f3f3f0;margin:24px}h1,h2{line-height:1.3}h2 small{font-size:12px;color:#5c5c56}a{color:#a84308;display:inline-block;margin:3px 10px 3px 0}section{background:white;border:1px solid #d2d2cb;border-radius:12px;padding:16px;margin:24px 0}.table{overflow:auto}table{border-collapse:collapse;font-size:13px;min-width:100%}th,td{border:1px solid #d2d2cb;text-align:left;padding:8px;vertical-align:top;white-space:pre-wrap}th{background:#ffead9}th small{display:block;font-size:11px;font-weight:normal;color:#5c5c56}td{max-width:600px;overflow-wrap:anywhere}pre{white-space:pre-wrap;overflow-wrap:anywhere;min-width:220px}summary{cursor:pointer;color:#963b08}nav{background:white;padding:12px;border-radius:10px}@media print{body{background:white;margin:8mm}section{break-before:page}.table{overflow:visible}details:not([open]){display:block}}</style></head><body><h1>Reifenapp 2.0 · Datenbanksicherung</h1>`
const backupReadme = `Reifenapp 2.0 – vollständige Datenbanksicherung

ZIP entpacken und uebersicht.html im Browser öffnen. Dafür wird keine
SQLite-Software benötigt. daten.json und schema.sql sind Textdateien.

datenbank.sqlite ist die vollständige, geprüfte und eigenständig nutzbare Kopie
der Datenbank einschließlich Kundendaten, Fahrzeugen, Radsätzen, Lagerorten,
Reservierungen, Historie, PDF-Vorlagen/Logos und Admin-Passwort-Hash.
Datenbank, HTML und JSON gehören zum selben konsistenten Datenstand.
Der Passwort-Hash wird in HTML/JSON ausgeblendet, bleibt in SQLite vollständig.
Die Sicherung ist nicht verschlüsselt und gehört nicht in ein öffentliches Repo.

manifest.json nennt Quelldatenbank, Zeitpunkt, Schemaversion, Zeilenzahlen
und SHA-256-Prüfsumme der Datenbankdatei. ZIP enthält zusätzlich CRC-Prüfsummen.
receipt-header.json enthält den im Server geladenen Firmenkopf.

Für eine Wiederherstellung einen neuen, leeren Projektordner verwenden und
den Server vor dem Kopieren stoppen. Keine vorhandene Datenbank ersetzen.
datenbank.sqlite als data/reifenapp.sqlite ablegen und mit go run . starten.
Bei Wiederherstellung der getrennten Entwicklungsdatenbank stattdessen
data/development.sqlite und go run . -dev verwenden. Nicht beide Daten mischen.
receipt-header.json bei Bedarf unter local/receipt-header.json ablegen.
Die App-Version muss das gesicherte Schema unterstützen. Erforderliche
Schema-Upgrades nur mit dem dokumentierten -migrate-Verfahren ausführen.
Das Admin-Passwort entspricht dem Passwort zum Sicherungszeitpunkt.

Eine automatische Wiederherstellung oder ein Import des JSON-Exports ist
nicht Teil dieser Downloadfunktion. Die laufende Datenbank wird nicht verändert.
`
