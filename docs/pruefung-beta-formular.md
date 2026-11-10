# Beta: Formularänderungen, 10. November 2026

Erfolgreich unter Linux amd64 mit Go 1.25.4:

- `go test ./...`, `go test -race ./...`, `go vet ./...`, Build.
- Zusätzliche Tests: Pflichtwerte beider Achsen einschließlich Profiltiefen,
  21 Profiltiefenwerte von 0 bis 10 mm, Bereichsgrenzen und ungültige Schritte,
  Kennzeichenformat, Kürzelliste ohne Duplikate und regionale Reihenfolge.
- Playwright/System-Chromium: Dropdownliste, Pflichtfelder, Übernahme aller fünf
  Reifengrößenwerte ohne Änderung einer bereits eingetragenen HA-Profiltiefe,
  Zurück und X mit Erhalt der Eingaben, vollständiges Speichern und Suche nach
  dem zusammengesetzten Kennzeichen, Mobilbreite 390px ohne Formularüberlauf.
- Screenshot der geschlossenen Startseite bei 1280×1000 bytegleich zu main.
- `git diff --check`.

Alle Tests ausschließlich mit fiktiven Daten in temporären/externen
Arbeitsverzeichnissen. Keine vorhandene Datenbank geändert; keine Migration nötig.
Nicht geprüft: echte CachyOS-/Pi-Hardware, LAN, weitere Browser und behördliche
Zuteilbarkeit jeder einzelnen Kennung. Die Kürzelliste ist ein lokaler Stand;
Quelle und Umfang stehen in kennzeichen.md.
