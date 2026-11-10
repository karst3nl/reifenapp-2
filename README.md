# Reifenapp 2.0

Lokale Reifenlagerverwaltung mit Go, SQLite und HTML/CSS/Vanilla JavaScript.
Die Original-README enthielt ausschließlich den Repository-Namen `reifenapp-2`.

## Installation und Start auf CachyOS

Voraussetzungen: Git, Go ab 1.23 und C-Compiler (SQLite-Treiber benötigt CGO).

```sh
sudo pacman -S --needed git go base-devel
git clone https://github.com/karst3nl/reifenapp-2.git
cd reifenapp-2
git switch feat/milestone-1
go mod download
go run .
```

Für das private Repository ist eine GitHub-Anmeldung per Git-Credential-Manager
oder SSH nötig. Im Browser http://127.0.0.1:8080 öffnen.

`go run .` erzeugt eine leere lokale Datenbank `data/reifenapp.sqlite`.
`go run . -dev` verwendet ausschließlich `data/development.sqlite` und fügt
bei deren erstmaliger Erstellung klar fiktive Beispiele ein. Bestehende Dateien
werden nicht neu befüllt oder zurückgesetzt. Entwicklungsdaten niemals in die
produktive Datenbank importieren. Laufzeitdaten bleiben durch `.gitignore` lokal.

## Erster Meilenstein

- Originalstartseite mit unverändertem CSS, SVG-Icons und responsivem Layout.
- Direkte Suche über `GET /api/search?q=...` nach Kunde, Kennzeichen oder Platz.
- JSON-Array mit `place`, `name`, `plate`, `car`, `date`; neueste Aktivität zuerst.
- Leerzeichen und Bindestriche werden ignoriert, Groß-/Kleinschreibung ebenfalls
  (einschließlich deutscher Umlaute). Leerer Suchtext liefert `[]`.
- Maximal 200 Zeichen; HTTP 400 bei Überschreitung, HTTP 500 bei Datenbankfehler.
- Ladezustand, keine Treffer und Fehleranzeige; sechs Vorgänge bleiben Platzhalter.
- 640 gültige Plätze: C1.1–200, C2.201–400, C3/C4/OL/OR jeweils 1–60.

## Struktur und Entscheidungen

```text
main.go                    Start, Flags und eingebettete Webdateien
internal/database/         SQLite, Initialisierung, Suche, Entwicklungsdaten
internal/handlers/         HTTP-API
internal/models/           Suchergebnis
web/index.html             Verbindliches Originaldesign
web/js/app.js              API-Anbindung und Platzhalterdialoge
migrations/                Versionierte SQL-Migrationen und Einbettung
docs/projektuebersicht.md   Bereitgestellte fachliche Grundlage
AGENTS.md                  Regeln für weitere Arbeiten
data/                      Erst beim Start erzeugt; nicht versioniert
```

CSS bleibt in der Designvorlage; ein leerer CSS-Ordner wäre unnötig.
Einziger direkter externer Go-Baustein ist der bewährte SQLite-Treiber
`github.com/mattn/go-sqlite3`. SQLite verwendet WAL, Fremdschlüssel und einen
Busy-Timeout. SQL-Migrationen laufen transaktional bei der Erstinitialisierung.
Bestehende Datenbanken werden auf die bekannte Schemaversion geprüft; unbekannte
Versionen werden abgelehnt. Vor späteren Migrationen bestehender Datenbanken
muss ein ausdrücklicher Backup-/Upgradeablauf ergänzt werden.

Belegung und Reservierung nutzen eine gemeinsame Tabelle `storage_claims`.
Ein partieller UNIQUE-Index schützt jeden Platz vor mehreren aktiven Ansprüchen,
auch bei konkurrierenden Schreibzugriffen und Statusänderungen. Ein aktiver
Radsatz darf ebenfalls nur einen Platz beanspruchen. Reservierungsformulare und
atomare Umwandlung sind noch nicht implementiert. Die Suche liest derzeit aktive
Ansprüche mit zugeordnetem Radsatz; keine vollständige Kunden- oder Historienansicht.
Die Suche filtert lokal in Go nach SQL-Abfrage, um Unicode ohne zusätzliche
SQLite-Erweiterungen korrekt zu behandeln; für größere Bestände später messen.

## Prüfungen

```sh
go test ./...
go test -race ./...
go vet ./...
go build -o bin/reifenapp .
```

Tests nutzen ausschließlich temporäre Datenbanken und fiktive Daten.

## Späterer Betrieb

Raspberry Pi: Go und C-Compiler auf dem Pi installieren, dort `go build` ausführen.
CGO erfordert beim Cross-Compiling eine passende C-Toolchain; zunächst auf dem
Zielgerät bauen. Dieser Betrieb ist noch nicht auf echter Pi-Hardware getestet.
Standardmäßig lauscht der Server ausschließlich auf Loopback. LAN-Zugriff ist
perspektivisch über `-addr 0.0.0.0:8080` möglich; vorher Werkstattnetz, Firewall und
Zugriffsschutz festlegen. Es gibt noch keine Authentifizierung oder TLS.
Keine Cloud für den laufenden Betrieb nötig. CSV, PDF, Administration, vollständige
Historie und Vorgangsformulare folgen später.
