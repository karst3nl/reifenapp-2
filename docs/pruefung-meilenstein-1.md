# Prüfung des ersten Meilensteins

10. November 2026, Linux amd64, Go 1.25.4.

Erfolgreich:

- `go test ./...`: Datenbank/WAL, 640 Plätze, Fremdschlüssel, Suche und Unicode,
  leere Ergebnisse, SQL-Sonderzeichen, Sortierung, Wiederöffnung ohne Datenverlust,
  kollidierende aktive Ansprüche, konkurrierende Reservierungen und HTTP-Fehler.
- `go test -race ./...`: keine gemeldeten Datenrennen.
- `go vet ./...` und `go build -o bin/reifenapp .`.
- Start aus dem Repository-Stamm mit `go run . -dev`.
- Playwright mit System-Chromium: echte API-Suche, keine Treffer, simulierte
  Netzwerkfehler, Platzhalterdialog und kein horizontaler Überlauf bei 390px.
- CSS und sämtliche SVG-Icons bytegleich mit der hochgeladenen Vorlage.
- `git diff --check`; SQLite, WAL/SHM und Builddateien werden ignoriert.

Der erste Browserversuch fand keinen Playwright-eigenen Browser; die Wiederholung
mit installiertem System-Chromium war erfolgreich.

Nicht ausgeführt: Betrieb auf echtem CachyOS-PC und Raspberry Pi, LAN-/Mehrgeräte-
Tests und vollständiger visueller Screenshotvergleich. Keine Tests für spätere,
noch nicht implementierte Formulare, Reservierungsumwandlung, CSV oder PDF.
