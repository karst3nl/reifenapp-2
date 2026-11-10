# Prüfung Neueinlagerung

10. November 2026; Linux amd64, Go 1.25.4, System-Chromium mit Playwright.
Ausschließlich fiktive Daten in temporären Testverzeichnissen bzw. unter dem
ignorierten Arbeitsverzeichnis außerhalb des Repositorys verwendet.
Bestehende lokale Datenbanken im Repository wurden nicht migriert oder verändert.

## Erfolgreich ausgeführt

- `go test ./...`, `go test -race ./...`, `go vet ./...`,
  `go build -o bin/reifenapp .`.
- Validierung: Pflichtfelder, Bereichsgrenzen aller sechs Bereiche, C2 ab 201,
  Dropdownwerte, Profiltiefenschritt, Aufbewahrungsort bei vorhandenem Zubehör.
- Transaktionale Speicherung mit Detaildaten und automatisch erzeugtem Zeitstempel;
  Suche nach Namen und Kennzeichen, Persistenz nach Wiederöffnung.
- Belegte und aktiv reservierte Plätze werden abgewiesen. Zwei parallele
  Einlagerungen über zwei unabhängige Datenbankverbindungen ergeben genau einen
  Erfolg und einen Konflikt. Ein Konflikt hinterlässt keine verwaisten Kunden,
  Fahrzeuge, Radsätze oder Details.
- Schema-1-Upgrade erfordert ausdrückliche Freigabe per Flag, erzeugt ein lesbares
  Backup mit ursprünglicher Schemaversion und erhält die beiden fiktiven Alteinträge.
  Ein provozierter Migrationsfehler rollt auch ALTER-TABLE-Änderungen zurück und
  lässt Schemaversion 1 sowie das Backup erhalten.
- HTTP: 201, 400, 403, 409, 415, 422, 500; fehlerhafte/zusätzliche JSON-Daten,
  Größenlimit, falscher Medientyp, fremder Ursprung, Suchintegration und Optionen.
- Browser: vollständige Eingabe einschließlich beider Achsen, Profiltiefen, aller
  Zubehörarten, Monteur und Bemerkungen; Erfolg, Suche, Konflikt mit Erhalt der
  Eingaben, C2-Bereichsprüfung, Formular bei 390px Breite ohne Überlauf.
- Tatsächlicher Serverneustart: Browserdatensatz danach weiter über API auffindbar;
  alle Detailfelder in der ausschließlich fiktiven Testdatenbank verifiziert.
- Screenshot der geschlossenen Startseite bei 1280×1000 bytegleich zu `origin/main`.
- `git diff --check`; Datenbank, WAL/SHM, Backups, Geheimnisse und Builddateien
  werden weiterhin ignoriert. Keine neue Go-Abhängigkeit.

## Nicht ausgeführt

Betrieb auf echter CachyOS-/Raspberry-Pi-Hardware, LAN-Mehrgerätebetrieb,
Lasttests und Browsermatrix außerhalb Chromium. Noch nicht implementierte
Reservierungsumwandlung, Kundenwiederverwendung, Administration und weitere
Vorgangsformulare wurden nicht als fertige Funktionen getestet.
