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

## Durchsuchbare Historie – 11. November 2026

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntaxprüfung und `git diff --check`.
Datenbanktests: Reservierung → Radwechsel → Auslagerung mit drei getrennten
Ereignissen, unveränderter alter Fahrzeug-Schnappschuss nach Modelländerung,
Detailwerte im JSON, persistente Ereignisse nach Wiederöffnung, freier Platz und
erneute Belegung bei weiterhin auffindbarer Historie, SQL-Änderungs-/Löschschutz,
Rollback aller vier Vorgangsarten bei absichtlich gescheiterter Historienanlage.
Schema-4-Upgrade: ältere Auslagerung wird suchbar, Platz bleibt frei, lesbares
Schema-4-Backup, keine doppelten Übernahmen nach Wiederöffnung. API-Test findet
Auslagerung nach Kennzeichen, Platz und Vorgangsbegriff.
Chromium: ursprüngliches Fahrzeug bei Reservierung erhalten, Radwechsel und
Auslagerung mit korrekten Bezeichnungen/Zuständen/Datum, Auslagerung über Formular,
Suchtext unverändert und Ergebnisse automatisch aktualisiert, nach Neuladen weiter
auffindbar, Platz frei und neu belegbar, alte und neue Ereignisse gemeinsam in der
Platzsuche, Begriffssuche „Auslagerung“, 390px ohne horizontalen Überlauf.
Nur fiktive temporäre Datenbanken. Keine bestehende Repository-Datenbank migriert.
Echte CachyOS-/Pi-Hardware, LAN und weitere Browser nicht geprüft.

## Automatische PDF-Belege – 11. November 2026

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntaxprüfung und `git diff --check`.
Go-PDF-Tests: alle vier Vorgangsarten, eine A4-Seite samt eingebetteter Schrift,
Maximallängen einschließlich 2000 Zeichen Bemerkungen ohne Kürzung, Reservierung
mit ausschließlich relevanten Daten, Altbestand und ungültige Nutzdaten.
HTTP-Tests: pdf_url aller vier Speicherantworten, echte PDF-Ausgabe, Inline-/No-Store-
Header, ungültige IDs/Vorgänge, unbekannte Belege, Datenbankfehler.
Chromium: echte automatische neue PDF-Tabs nach allen vier Formularen,
keine Tabs bei Clientvalidierung, vorbereiteter Tab bei Speicherfehler geschlossen,
PDF-Links in der Historie, manuell simulierter Popup-Blocker mit funktionierendem
Fallback-Link. Suchtext und bestehendes Formularverhalten erhalten.
Zusätzlich pdfinfo: normale und maximale Texte genau eine Seite A4
(595.28×841.89 pt). pdftotext: Umlaute, Škoda, Profiltiefen, Zubehör, ENDMARKER und
sämtliche Zeichen der Maximaltexte erhalten. Normale und maximale Beispielseiten
als Bilder gerendert und visuell auf Überlauf/Abschneiden geprüft.
Alle Beispiele ausschließlich fiktiv unter externem Arbeitsverzeichnis.
Kein Schemawechsel. Echter Drucker, CachyOS-/Pi-Hardware, LAN und weitere Browser
nicht geprüft. Browser-/Druckereinstellungen können die PDF-Anzeige beeinflussen.

## PDF-Layout nach HTML-Druckvorlage – 11. November 2026

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntax und `git diff --check`.
Neue Tests für lokalen Firmenkopf (fehlende/gültige/fehlerhafte Datei,
Längenprüfung), alle vier PDF-Arten mit Firmenkopf und die Reihenfolge der
zweispaltigen Stammdaten sowie dreispaltigen Reifen-/Zubehörblöcke.
Bestehende Tests für genau eine A4-Seite und vollständige Maximaltexte bleiben grün.
PDFs mit fiktivem vierzeiligem Firmenkopf erzeugt, mit pdfinfo als eine A4-Seite
bestätigt, via pdftotext vollständig einschließlich Umlaute/Maximaltext-Endmarkierung
geprüft sowie gerenderte normale und maximale Belege visuell geprüft.
Chromium-Regressionsprüfung: alle vier automatischen PDF-Tabs, keine Tabs bei
ungültigen Eingaben, Schließen des vorbereiteten Tabs bei Speicherfehler,
Historien-Nachdruck und Ersatzlink bei simuliertem Popup-Blocker erfolgreich.
Reale Firmenangaben aus dem Anhang wurden nicht versioniert. Keine Datenbanken
verändert oder migriert. Echter Drucker, CachyOS-/Pi-Hardware und andere Browser
nicht geprüft. Alte Reifensuch-App-Repositories wurden nicht verwendet.

## Admin-PDF-Designer – 12. November 2026

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntax und `git diff --check`.
Neue Tests: gültige einseitige A4-Vorlagen für vier Vorgangsarten, Formen/Farben,
Seitenbegrenzung, unbekannte Felder, falsche Größen/Farben, doppelte IDs,
Textüberlauf und vollständiger Standard-Fallback. Lokale Vorlagen bleiben nach
Wiederöffnung erhalten; Revisionskonflikte werden abgewiesen, Vorgangsarten bleiben
getrennt, fehlerhafte Vorlagen werden nicht gespeichert. Upgrade Schema 5 auf 6
benötigt Freigabe, erhält beide historischen Alteinträge und erstellt ein lesbares
Schema-5-Backup.
Admin-API: ohne Login gesperrt, falsches Passwort, HttpOnly-/SameSite-Cookie,
CSRF-Prüfung, Layout laden/speichern, echte PDF-Vorschau, Revisionskonflikt,
Vorschauwerte, Schriftabruf, Logout, Sitzungsablauf, deaktivierter Admin ohne
Passwort und Begrenzung der Anmeldeversuche.
Chromium: Navigation über Administration, Anmeldung, echtes Ziehen und Skalieren,
Eigenschaften, Duplizieren/Rückgängig/Wiederholen, Vorschau-PDF als Browser-Blob,
Aktivieren und Neuladen, eigener Textmarker auch im echten Beleg, getrennte
Vorgangsarten, Zurückschalten auf Standard, 390px ohne Seitenüberlauf, Logout.
PDF-Textprüfung und pdfinfo bestätigen Marker in Vorschau und eigenem Echtbeleg,
keinen Marker nach Standardumschaltung und genau eine A4-Seite. Zu kleines
aktives Namensfeld wurde mit 200 Zeichen realer fiktiver Eingabe getestet:
vollständiger Standardbeleg mit sichtbarem Fallback-Hinweis.
Nur fiktive temporäre Datenbanken und Testpasswort, keine vorhandenen lokalen
Repository-Datenbanken migriert. Keine neue Abhängigkeit. Echte CachyOS-/Pi-
Hardware, LAN-Betrieb, Drucker und weitere Browser nicht geprüft.

## Einfacher Admin-Login mit Ersteinrichtung – 12. November 2026

Erfolgreich: `go test ./...`, `go test -race ./...`, `go vet ./...`, Build,
JavaScript-Syntaxprüfung und `git diff --check`.
Datenbanktests: Startpasswort nur als bcrypt-Hash, verpflichtender Wechsel,
Passwortrichtlinie, neues Passwort überschreibt den Hash, bestehendes Konto
wird beim Start nicht zurückgesetzt, falsches aktuelles Passwort und veraltete
Revision abgewiesen. Migration Schema 6 auf 7 erhält Kunden, Historie und
PDF-Vorlage; lesbares Backup mit Schemaversion 6.
API: fester Benutzer, Startanmeldung nur mit eingeschränkter Sitzung,
Designer-/Feld-/Schriftzugriff vor Passwortwechsel gesperrt, CSRF und Bestätigung,
Sitzungswechsel, alte Sitzung ungültig, Startpasswort nach Änderung abgewiesen,
eigenes Passwort akzeptiert, spätere Änderung erfordert aktuelles Passwort,
Rate-Limit und Sitzungsablauf. Nach erneutem Öffnen der Datenbank und echter
Serverneustart: neues Passwort bleibt gültig, Startpasswort ungültig.
Chromium: vorbelegter Benutzer admin, Pflichtdialog auch nach Seitenneuladen,
abweichende Wiederholung und erneute Startpasswortwahl abgewiesen, erfolgreicher
Wechsel öffnet Designer, Felder danach leer, Logout und neues Login,
regulärer Passwortwechsel und PDF-Vorlage unverändert. Echten Server ohne
Umgebungsvariable neu gestartet und eigene Anmeldung mit erhaltener Vorlage geprüft.
Nur fiktive Testpasswörter/-daten in separaten temporären Datenbanken.
Keine vorhandene Repository-Datenbank migriert. Echte CachyOS-/Pi-Hardware,
LAN, Drucker und andere Browser nicht geprüft.

## Raster im PDF-Designer – 12. November 2026

Erfolgreich: JavaScript-Syntax, `go test ./...`, Build und `git diff --check`.
Chromium: sichtbares Standardraster, 2/5/10-mm-Ausrichtung beim echten Ziehen,
Skalieren an rechten/unteren Rasterkanten, Pfeiltasten mit/ohne Umschalt, neue und
duplizierte Elemente, freie Bewegung bei „Aus“/Alt, manuelle Präzisionswerte,
Rückgängig/Wiederholen, Zoom 50/75/100 %, Begrenzung auf A4, Speichern/Neuladen
und Mobilbreite. Die PDF-Vorschaudaten sind vor/nach Rasterumschaltung identisch;
das sichtbare Raster gelangt nicht in die PDF-Vorlage. PDF-Vorschau weiter gültig.
Keine Backend-/Schemaänderung; Race-Tests für diesen Frontend-Schritt nicht erneut
ausgeführt. Nur fiktive temporäre Daten. Hardware, LAN, Drucker und andere Browser
nicht geprüft.
