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

## Vollbild und Speichern

Ergänzende Browserprüfung: erfolgreicher POST schließt den Dialog und erhält einen
vorher eingetragenen Suchtext. Bestätigung außerhalb des Dialogs sichtbar.
Normales Formular bei 1440×900 und 1366×768 ohne vertikales Scrollen; bei 390px
kein horizontaler Überlauf. Simulierter HTTP-409-Fehler lässt den Dialog offen.
`go test ./...`, Build und `git diff --check` erfolgreich. Keine Backendänderung;
Race-Tests wurden für diese reinen HTML-/CSS-/JavaScript-Änderungen nicht erneut
 ausgeführt. Hardware-/LAN- und weitere Browserprüfungen weiterhin nicht ausgeführt.

## Namensvorschläge und Formularzustand

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntaxprüfung und `git diff --check`.
Zusätzliche API-Tests für Unicode, unterschiedliche Groß-/Kleinschreibung,
Duplikatvermeidung, maximal 20 Vorschläge, keine Treffer, leere Anfrage,
Sonderzeichen, Überlänge, Datenbankfehler und No-Store-Header.
Browserprüfung mit System-Chromium: vorhandener fiktiver Namensvorschlag,
Pflichtfeldmarkierung bei Verlassen und Speicherversuch, Entfernen nach Korrektur,
komplettes Leeren nach X/Zurück/Schließen/Escape einschließlich Zubehör und
Vorschlägen, freie Namenseingabe bei ausgefallener Vorschlags-API.
Keine Migration und keine neue Abhängigkeit. Hardware-/LAN- und weitere
Browserprüfungen weiterhin nicht ausgeführt.

## Eigene Vorschlagsliste und Platzkonflikte

Erfolgreich: `go test ./...`, Build, JavaScript-Syntaxprüfung, `git diff --check`.
Browserprüfung mit Chromium: Tab übernimmt und bewegt Fokus weiter, Pfeiltasten/
Enter, Mausklick, Escape schließt nur die Liste, Zurücksetzen beim Schließen,
Mobilbreite 390px. Tatsächlich belegter fiktiver Platz C1.25 liefert HTTP 409:
Nummernfeld erhält roten 2px-Rahmen und Fokus, Konflikt bleibt nach Fokuswechsel
bestehen und verschwindet nach Änderung der Nummer. Keine neue Migration oder
Backendänderung. Race-/Hardware-/LAN-Tests und weitere Browser für diesen reinen
Frontend-Schritt nicht erneut ausgeführt.

## Direkte Platzprüfung und Reservierungsabgleich

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntaxprüfung und `git diff --check`.
Neue Datenbanktests: frei, belegt, reserviert, unbekannter Platz, fehlende
Zuordnung, falscher Name, falsches Kennzeichen, normalisierter exakter Abgleich,
passende Umwandlung ohne doppelte Kunden/Radsätze, Suchintegration,
Rollback bei absichtlich ausgelöstem Detailfehler, genau eine erfolgreiche
Umwandlung bei parallelem Zugriff über unabhängige Verbindungen.
API-Tests: Statusantworten, ungültige Felder/JSON, Datenbankfehler, No-Store,
keine Offenlegung des Kunden am belegten Platz.
Browserprüfung: unmittelbare Anzeige belegt/frei/reserviert, falsches Kennzeichen
sperrt, beide passenden Identitätsangaben erlauben Speicherung, tatsächliche
Reservierungsumwandlung mit anschließender Suche, erneute Prüfung zeigt Belegung,
Netzwerkfehler sperrt und erneute Prüfung funktioniert, Schließen leert Anzeige.
Nur fiktive Reservierungen in einer separaten temporären Testdatenbank eingesetzt.
Keine vorhandenen lokalen Repository-Datenbanken verändert. Keine Migration.
CachyOS-/Pi-Hardware, LAN und andere Browser weiterhin nicht getestet.

## Reservierungsformular

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntaxprüfung und `git diff --check`.
Neue Datenbank-/API-Tests: Reservierung ohne Größen/Zubehör anlegen, Wiederöffnung
mit erhaltenen Daten, Suche mit Typkennzeichnung, Doppelreservierung einschließlich
gleicher Identität ablehnen, belegten Platz ablehnen, gemeinsame Pflichtfeldprüfung,
unzulässige Zusatzfelder ablehnen, Rollback aller Teilschritte bei Speicherfehler,
genau eine erfolgreiche Reservierung bei parallelen unabhängigen Verbindungen,
Unterschied zwischen Reservierungsprüfung und Einlagerungsabgleich, anschließende
Umwandlung mit Wiederverwendung des Kunden und Radsatzes.
Chromium-Browserprüfung: reduzierte sichtbare Felder, Pflichtfeldmarkierungen,
Namensvorschlag mit Tab, direkte Platzprüfung, Speichern/automatisches Schließen,
Suchtreffer mit „Reserviert“, gleiche Identität ermöglicht keine Doppelreservierung,
Mobilbreite 390px, Leeren beim Öffnen, Wechsel zur vollständigen Neueinlagerung
mit allen Pflichtfeldern und tatsächliche Umwandlung bis zum Suchtreffer.
Nur fiktive Testdaten in separaten temporären Datenbanken. Keine Migration.
Echte CachyOS-/Pi-Hardware, LAN und weitere Browser nicht geprüft.

## Reservierung ohne Reifen-/Felgenart – 11. November 2026

Reifenart/Felgenart sind im Reservierungsmodus ausgeblendet und deaktiviert;
Reservierungs-API und Validierung benötigen beide nicht mehr. Die Neueinlagerung
benötigt sie weiterhin. Neue Reservierungen erzeugen keine leeren Detaildatensätze.
Bestehende Reservierungsdaten werden nicht verändert. Keine Migration.
„Erneut prüfen“ hat 6px zusätzlichen oberen Abstand.

Erfolgreich: Go-Tests, Race-Tests, go vet, Build, JavaScript-Syntax und diff-check.
Chromium: Formular ohne die entfernten Felder speichern, Treffer als reserviert,
Doppelreservierung blockieren, Mobilbreite, danach in vollständige Neueinlagerung
wechseln und konvertieren. Abstand zwischen Nummerneingabe und Prüfbutton anhand
der Elementpositionen geprüft. CachyOS-/Pi-Hardware, LAN und weitere Browser
nicht getestet.

## Radwechsel – 11. November 2026

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntaxprüfung und `git diff --check`.
Neue Datenbanktests: Radwechselkennung persistent nach Wiederöffnung,
Suchantwort mit operation, Neueinlagerung weiterhin storage, Pflichtfelder,
Doppelbelegung, passende Reservierung umwandeln, Upgrade Schema 2 auf 3 mit
lesbarem Schema-2-Backup und Erhalt alter Detaildaten, SQL-Constraint für Vorgangstyp.
API-Tests: eigener Endpunkt, 201 mit wheel_change, 409, 422, 405, 500,
Ablehnen eines vom Client vorgegebenen operation-Feldes.
Chromium: Vollformular, Pflichtfeldmarkierungen, belegter Platz, Kopierbutton ohne
Profiltiefe, tatsächlicher POST /api/wheel-changes, automatisches Schließen,
Suchtext unverändert, Suchtreffer als Radwechsel, Reset und Mobilbreite sowie
Wechsel zu Reservierung und Neueinlagerung mit korrekten Feldgruppen.
Nur fiktive temporäre Daten verwendet. Keine vorhandenen Repository-Datenbanken
migriert. Echte CachyOS-/Pi-Hardware, LAN und weitere Browser nicht geprüft.

## Auslagerung – 11. November 2026

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntax und `git diff --check`.
Datenbank-/API-Tests: Zielbelegung samt Kunden-/Fahrzeugdaten lesen, Bestätigung
prüfen, Belegung beenden und Platz freigeben, Einlagerungsdaten und ursprünglichen
Zeitstempel erhalten, Auslagerungszeitpunkt protokollieren, nach Wiederöffnung frei,
erneut einlagern, veraltete Kennung ohne Freigabe der neuen Belegung abweisen,
Reservierung und ungültigen Platz abweisen, Rollback bei Protokollfehler,
genau ein Erfolg bei konkurrierenden unabhängigen Verbindungen, passende HTTP-
Fehlercodes und No-Store. Schema-3-Upgrade erhält beide Alteinträge und erzeugt
ein lesbares Backup mit Schemaversion 3; neue Auslagerung funktioniert danach.
Chromium: aktive Belegung anzeigen, Bestätigungscheckbox, reale Auslagerung,
aktualisierte Suche mit unverändertem Text, freier Platz erneut belegbar,
zwischenzeitlicher Austausch durch einen anderen Zugriff erzeugt Konflikt und
lässt die neue Belegung unverändert, erneute Prüfung, Reservierung/freier Platz
nicht auslagerbar, Mobilbreite 390px und Formularreset.
Nur fiktive temporäre Testdatenbanken. Keine vorhandene Repository-Datenbank
migriert. Echte CachyOS-/Pi-Hardware, LAN und andere Browser nicht geprüft.
