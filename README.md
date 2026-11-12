# Reifenapp 2.0

Open-Source-Reifenlagerverwaltung für Kfz-Werkstätten mit Go, SQLite und
HTML/CSS/Vanilla JavaScript. Lokal betreibbar, ohne Cloudpflicht.

**Lizenz:** [MIT](LICENSE) · **Entwicklung:** [beta](https://github.com/karst3nl/reifenapp-2/tree/beta)

![Aktuelle Startseite der Reifenapp](docs/images/startseite.png)

Aktueller Entwicklungsstand auf [`beta`](https://github.com/karst3nl/reifenapp-2/tree/beta).

## Installation und Start auf CachyOS

Voraussetzungen: Git, Go ab 1.23 und C-Compiler (SQLite-Treiber benötigt CGO).

```sh
sudo pacman -S --needed git go base-devel
git clone https://github.com/karst3nl/reifenapp-2.git
cd reifenapp-2
git switch beta
go mod download
go run .
```

Das Repository ist öffentlich. Zum Herunterladen ist keine GitHub-Anmeldung nötig.
Im Browser http://127.0.0.1:8080 öffnen.

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
- Ladezustand, keine Treffer und Fehleranzeige; Radwechsel, Neueinlagerung, Reservierung und Auslagerung sind nutzbar; Die Historie ist über die Suche zugänglich; Administration enthält den PDF-Designer.
- 640 gültige Plätze: C1.1–200, C2.201–400, C3/C4/OL/OR jeweils 1–60.

## Struktur und Entscheidungen

```text
main.go                    Start, Flags und eingebettete Webdateien
internal/database/         SQLite, Initialisierung, Suche, Entwicklungsdaten
internal/handlers/         HTTP-API
internal/models/           Suchergebnis
web/index.html             Verbindliches Originaldesign
web/js/app.js              Suchanbindung und übrige Platzhalterdialoge
web/js/storage.js          Neueinlagerungsformular
web/css/storage.css        Ausschließlich auf das neue Formular begrenztes CSS
migrations/                Versionierte SQL-Migrationen und Einbettung
docs/projektuebersicht.md   Bereitgestellte fachliche Grundlage
AGENTS.md                  Regeln für weitere Arbeiten
data/                      Erst beim Start erzeugt; nicht versioniert
```

CSS bleibt in der Designvorlage; ein leerer CSS-Ordner wäre unnötig.
Direkte externe Go-Bausteine sind der SQLite-Treiber
`github.com/mattn/go-sqlite3`, für lokale PDF-Belege `github.com/go-pdf/fpdf` und
für Passwort-Hashes `golang.org/x/crypto/bcrypt`. SQLite verwendet WAL, Fremdschlüssel und einen
Busy-Timeout. SQL-Migrationen laufen transaktional bei der Erstinitialisierung.
Bestehende Datenbanken werden auf die bekannte Schemaversion geprüft; unbekannte
Versionen werden abgelehnt. Schema 1 bis 6 benötigen das unten beschriebene explizite
Upgrade mit Integritätsprüfung und konsistentem SQLite-Backup.

Belegung und Reservierung nutzen eine gemeinsame Tabelle `storage_claims`.
Ein partieller UNIQUE-Index schützt jeden Platz vor mehreren aktiven Ansprüchen,
auch bei konkurrierenden Schreibzugriffen und Statusänderungen. Ein aktiver
Radsatz darf ebenfalls nur einen Platz beanspruchen. Das Reservierungsformular ist implementiert. Die atomare Umwandlung einer
eindeutig zugeordneten, passenden Reservierung ist im Neueinlagerungsablauf möglich. Die Suche liest die Vorgangshistorie mit unveränderlichen Schnappschüssen;
eine separate vollständige Kunden-/Historienansicht folgt später.
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
Historie und weitere Vorgangsformulare folgen später.

## Neueinlagerung (zweiter Meilenstein)

### Vorhandene Installation aktualisieren

Alle laufenden Reifenapp-Server zuerst im jeweiligen Terminal mit Strg+C beenden.
Im bestehenden Repository (lokale Änderungen vorher mit `git status` prüfen):

```sh
git fetch origin
git switch beta
git pull --ff-only origin beta
go run . -dev -migrate
```

Diese Variante arbeitet weiterhin mit den fiktiven Entwicklungsdaten.
Für die normale Datenbank stattdessen `go run . -migrate` verwenden.
Einmalig nötig bei Schema 1 bis 6: Integritätsprüfung, dann `VACUUM INTO` für eine
konsistente Sicherung inklusive noch im WAL befindlicher Änderungen, danach
transaktionale Migrationen bis einschließlich 007. Das Backup liegt neben der ursprünglichen Datei
als `*.sqlite.backup-<UTC-Zeitstempel>` und ist ebenfalls ignoriert.
Bei einem Backupfehler findet keine Migration statt. Vorhandene Datensätze werden
nicht neu angelegt, gelöscht oder mit Beispielen überschrieben. Alle anderen
Prozesse mit Zugriff auf diese Datenbank müssen während des Upgrades beendet sein.
Anschließend genügt wieder `go run . -dev` bzw. `go run .`.
Keinen älteren App-Stand auf einer bereits aktualisierten Datenbank starten.

### Im Browser testen

1. http://127.0.0.1:8080 öffnen und „Neueinlagerung“ anklicken.
2. Einen freien Platz wählen, etwa C1.2, und „Fiktiver Testkunde“ mit
   Kennzeichen „MZ-NE 123“, Marke/Modell, Reifen- und Felgenart erfassen.
3. Beide Achsen und Profiltiefen ausfüllen; optional Zubehör, Monteur und Bemerkungen ergänzen; speichern.
4. Bestätigung mit Lagerplatz prüfen, Dialog schließen und nach Name oder
   Kennzeichen suchen. Nach einem Serverneustart bleibt die Einlagerung vorhanden.
5. Derselbe Platz wird beim erneuten Speichern als belegt abgewiesen. Auch eine
   aktive Reservierung sperrt den Platz, solange Name und Kennzeichen nicht beide passen.

### Annahmen und Pflichtfelder

Die Projektübersicht legt keine abschließende Pflichtfeldliste und keine konkreten
Dropdownwerte fest. Für diesen Meilenstein gelten ausdrücklich diese Annahmen:

- Pflicht: Lagerbereich/-nummer, Kundenname, Kennzeichen, Marke, Modell,
  Reifenart und Felgenart. Sie sind im Formular mit * gekennzeichnet.
- Optional: Zubehör, Monteur,
  Bemerkungen. Die fünf Reifengrößenangaben und die Profiltiefe sind für beide Achsen Pflicht.
- Falls Zubehör als vorhanden markiert ist, wird dessen Aufbewahrungsort Pflicht.
  Bei nicht vorhandenem Zubehör wird kein Aufbewahrungsort gespeichert.
- Die anfänglichen Dropdownlisten sind in `internal/models/storage.go` zentral
  hinterlegt und über die Optionen-API verfügbar. Noch keine Adminpflege.
  Marke „Sonstige“ ist für nicht gelistete Hersteller verfügbar.
- Größenlisten: Breite 125–355 mm (10er-Schritte), Querschnitt 20–85 (5er-Schritte),
  Zoll 10–24, Traglastindex 50–130; gängige Geschwindigkeitsindizes einschließlich
  `(Y)`. Diese Auswahl ist eine erste Eingabehilfe, keine technische
  Zulassungs-/Kompatibilitätsprüfung und kein vollständiger Reifenkatalog.
- Profiltiefe verpflichtend 0–10 mm in 0,5-mm-Schritten, ohne Verschleißbewertung.
- Jede Neueinlagerung erstellt vorerst einen eigenen Kunden, ein Fahrzeug und
  einen Radsatz. Vorhandene Kunden werden nicht automatisch anhand ihres Namens
  zusammengeführt. Kundenauswahl/-zusammenführung folgt später.
- Neue Einlagerungen erhalten serverseitig UTC-Zeitstempel. Bestehende fiktive
  Beispiele behalten ihre Kennzeichnung; neue Testnamen bitte mit „fiktiv“ versehen.

### API und Datenmodell

`GET /api/storage/options` liefert JSON-Listen für die Dropdowns.
`POST /api/storage` erwartet `Content-Type: application/json` mit:

```json
{
  "area": "C1", "number": 2,
  "name": "Fiktiver Testkunde", "plate": "MZ-NE 123",
  "brand": "VW", "model": "Golf", "tire_type": "Winter", "rim_type": "Stahl",
  "front": {"width":"205","ratio":"55","inches":"16","load":"91","speed":"H"},
  "rear": {"width":"205","ratio":"55","inches":"16","load":"91","speed":"H"}, "front_depth":"6.5", "rear_depth":"6.0",
  "hubcaps":{"present":true,"location":"Am Radsatz"},
  "bolts":{"present":false,"location":""},
  "lock":{"present":false,"location":""},
  "mechanic":"Fiktiver Monteur", "notes":"Nur Testdaten"
}
```

HTTP 201: `{ "id": 3, "place": "C1.2", "message": "Neueinlagerung gespeichert." }`.
Fehler enthalten `error`, bei HTTP 422 zusätzlich eine `fields`-Zuordnung.
Weitere Statuscodes: 400 ungültiges JSON/unbekannte Felder/zu große Anfrage,
403 fremder Browserursprung, 409 Platz belegt oder aktiv reserviert,
415 falscher Content-Type, 500 interner Speicherfehler ohne Datenbankdetails.
Maximale Anfragegröße: 32 KiB. Die bestehende Such-API bleibt unverändert.

Migration `002_storage_details.sql` ergänzt Marke/Modell am Fahrzeug und
`storage_details` am Einlagerungsanspruch. Achsdaten und die drei festen
Zubehörangaben werden als JSON in dieser Detailtabelle gespeichert. Die kleine,
zusammengehörige Struktur benötigt vorerst keine weiteren Untertabellen.
Kunde, Fahrzeug, Radsatz, Platzanspruch und Details werden in **einer Transaktion**
gespeichert. Der vorhandene UNIQUE-Index ist die verbindliche Konfliktprüfung.
Fehler rollen alle Teilschritte zurück. JSON-Schreibzugriffe aus fremden
Browserursprüngen werden abgelehnt; LAN-Authentifizierung bleibt ein späteres Thema.

## Beta-Entwicklung und Formularänderungen

Weitere Entwicklung wird auf `beta` gesammelt. `main` wird nicht automatisch
aktualisiert; ein späterer Merge erfolgt gesondert. Zum Wechseln den Server stoppen,
`git fetch origin`, `git switch beta`, `git pull --ff-only origin beta` ausführen.
Start weiterhin `go run . -dev` (ein Upgrade von Schema 1 bis 6 benötigt einmalig zusätzlich
`-migrate`, siehe oben). Für diese Formularänderungen ist keine Migration nötig.

Kennzeichen: Landkreiskürzel per Dropdown, Buchstaben und Nummer separat;
Mainz und Umgebung stehen zuerst. Quelle und Umfang: `docs/kennzeichen.md`.
Beide Achsen benötigen Breite, Querschnitt, Zoll, Traglastindex,
Geschwindigkeitsindex und ihre eigene Profiltiefe (0–10 mm, 0,5-mm-Schritte).
„Reifendaten der Vorderachse übernehmen“ kopiert nur die fünf Größenangaben;
eine schon eingetragene Hinterachs-Profiltiefe bleibt unverändert.
Zurückpfeil und X schließen das Formular ohne Speichern und verwerfen die Eingaben. Während des Speicherns sind beide gesperrt.

### Vollbild und Abschluss der Neueinlagerung

Das Formular nutzt die gesamte Bildschirmfläche und ordnet die Abschnitte auf
breiten Bildschirmen nebeneinander an. Bei 1440×900 und 1366×768 ist der normale
Formularinhalt ohne Scrollen sichtbar. Kleinere Fenster, Vergrößerung und zusätzliche
Fehlermeldungen können Scrollen erfordern; alle Felder bleiben erreichbar.
Nach bestätigter Speicherung schließt das Formular und zeigt eine Bestätigung
auf der Startseite. Der vorhandene Suchtext bleibt unverändert; der neue Eintrag
wird nicht automatisch in die Suche eingesetzt. Bei Fehlern bleibt das Formular
offen und behält die Eingaben.

### Namensvorschläge, leeres Formular und Pflichtfeldmarkierung

`GET /api/customers?q=Ma` liefert bis zu 20 unterschiedliche vorhandene Namen
mit passendem Anfang, ohne Beachtung der Groß-/Kleinschreibung (auch Umlaute).
Leerer Suchtext liefert `[]`; maximal 200 Zeichen. Antworten werden mit
`Cache-Control: no-store` ausgeliefert. Der Vorschlag ergänzt nur den Namen;
Kunden werden weiterhin nicht automatisch zusammengeführt. Neue Namen sind frei
möglich, auch wenn die Vorschlagsabfrage fehlschlägt.

Beim Schließen über X, Zurück, Schließen oder Escape werden alle Formularwerte,
Zubehörzustände, Vorschläge und Fehlermarkierungen geleert. Auch das erneute Öffnen
startet leer. Keine Speicherung von Entwürfen in Browser-Storage; Browser-Autofill
ist für das Formular deaktiviert. Bereits dauerhaft gespeicherte Einlagerungen
und die unabhängige Suchleiste werden dabei nicht verändert.

Verlassene ungültige Pflichtfelder werden mit rotem Rahmen, Hintergrund und
Texthinweis markiert. Beim Speicherversuch werden alle fehlenden/ungültigen Felder
markiert und das erste fokussiert. Korrigierte Felder verlieren ihre Markierung.
Serverseitige Validierung bleibt zusätzlich aktiv. Ein Speicherfehler erhält die
Eingaben nur solange das Formular geöffnet bleibt.

### Namensauswahl und belegte Plätze

Die Vorschläge erscheinen in einer eigenen Liste im App-Design, ohne natives
Browser-Datalist. Der erste Treffer ist markiert; Pfeil hoch/runter ändert die
Auswahl. Tab übernimmt den markierten Namen und wechselt ins nächste Feld;
Enter oder Mausklick übernimmt ebenfalls. Escape schließt zuerst die Vorschläge.
Freie Namen bleiben möglich; Shift+Tab übernimmt keinen Vorschlag.

HTTP 409 bei belegtem oder reserviertem Platz zeigt einen auffälligen roten
Fehlerkasten und einen roten Rahmen mit direktem Hinweis am Nummernfeld.
Das Nummernfeld erhält den Fokus. Der Konflikthinweis bleibt auch nach Verlassen
des Feldes bestehen und wird bei Änderung von Lagerbereich oder Nummer entfernt.

### Direkte Platzprüfung und passende Reservierungen

Beim Ändern von Bereich, Nummer, Name oder Kennzeichen wird der Platz nach kurzer
Eingabepause direkt geprüft. Frei erscheint grün; belegt oder nicht passend
reserviert erscheint rot am Nummernfeld. Während einer laufenden oder gescheiterten
Prüfung sowie bei einem Konflikt ist Speichern gesperrt. „Erneut prüfen“ wiederholt
die Abfrage, etwa nach einem Verbindungsfehler oder einer zwischenzeitlichen Änderung.
Die Vorschau ist keine Platzsperre: Beim Speichern wird stets erneut transaktional
geprüft; ein inzwischen belegter Platz wird weiterhin zuverlässig abgewiesen.

`POST /api/storage/check` erwartet JSON mit `area`, `number`, `name`, `plate` und
liefert `place`, `status` (`free`, `occupied`, `reserved`), `allowed`, `matches`,
`message`. HTTP 200 beschreibt auch regulär belegte/reservierte Plätze;
422 ungültige Prüfeingaben, 400 ungültiges JSON, 500 Prüfungsfehler.
Kein Cache und keine Offenlegung des Namens/Kennzeichens des Reservierenden.

Für reservierte Plätze müssen Kundenname **und** Kennzeichen übereinstimmen:
Name ohne Beachtung der Groß-/Kleinschreibung, mit vereinheitlichten Leerzeichen;
Kennzeichen zusätzlich ohne Leerzeichen/Bindestriche. Keine unscharfe Namenssuche
für die Freigabe. Der Zusammenhang ergibt sich aus der bestehenden Verknüpfung
Reservierung → Radsatz → Fahrzeug → Kunde. Reservierungen ohne zugeordneten
Kunden und Fahrzeug bleiben gesperrt. Das Reservierungsformular legt diese Zuordnung beim Speichern an;
für reale Daten keine manuellen SQL-Eingriffe als Teil dieses Ablaufs nötig.

Bei passender Reservierung markiert dieselbe SQLite-Transaktion diese als
`converted`, verwendet den zugeordneten Radsatz/Kunden/das Fahrzeug weiter,
aktualisiert Marke/Modell und speichert Belegung und erfasste Details. Ein Fehler
rollt auch die Reservierungsumwandlung zurück. Der alte Reservierungsdatensatz
bleibt erhalten. Keine neue Migration oder Abhängigkeit erforderlich.

## Reservierungsformular

„Reservierung“ öffnet dieselbe Formularoberfläche im reduzierten Modus.
Pflichtangaben: Bereich/Nummer, Kundenname, Kennzeichen und Fahrzeugmarke/Modell;
ohne Reifenart und Felgenart. Keine Größen, Profiltiefen, Zubehörangaben, Monteure oder
Bemerkungen. Dieselben Namensvorschläge mit Tab, regionales Kennzeichen-Dropdown,
roten Pflichtfeldhinweise, Vollbild, X/Zurück und Zurücksetzen beim Schließen.
Nach Erfolg schließt das Formular; der Suchtext bleibt unverändert.

Die Liveprüfung ist im Reservierungsmodus strenger: Nur **freie** Plätze sind
reservierbar. Belegte oder bereits reservierte Plätze bleiben gesperrt, auch bei
identischem Namen und Kennzeichen. Neue Reservierungen sind in der Suche mit
„Reserviert“ gekennzeichnet. Später „Neueinlagerung“ öffnen, denselben Platz,
Namen und dasselbe Kennzeichen sowie die vollständigen Reifendaten erfassen.
Der bestehende atomare Umwandlungsablauf verwendet die Zuordnung weiter.

`POST /api/reservations` erwartet JSON mit ausschließlich:

```json
{
  "area":"C1", "number":9, "name":"Reservierung Fiktiv", "plate":"MZ-RF 987",
  "brand":"VW", "model":"Golf"
}
```

201 liefert `id`, `place`, `message`; 422 Feldfehler, 409 belegter/reservierter
Platz, 400 ungültiges JSON/unerwartete Zusatzfelder, 415 falscher Medientyp,
403 fremder Ursprung und 500 Speicherfehler. Kunde, Fahrzeug, Radsatz und aktive
Reservierung werden transaktional mit serverseitigem Zeitstempel gespeichert.
Reservierungen benötigen keine Einträge in der Einlagerungs-Detailtabelle.
Bereits gespeicherte Angaben älterer Reservierungen bleiben unangetastet.
Keine erfundenen Reifendaten, keine neue Migration, keine neue Abhängigkeit.

`POST /api/storage/check` akzeptiert jetzt optional `purpose: "reservation"`.
Damit bleibt auch eine passende bestehende Reservierung für eine weitere
Reservierung gesperrt. Ohne Angabe bzw. mit `purpose: "storage"` gilt der
bestehende Einlagerungsabgleich. Die Suchantwort enthält zusätzlich `kind`
(`reservation` oder `occupancy`). Kein Ablaufdatum, automatische Freigabe oder
Stornierungsformular in diesem Schritt.

## Radwechsel

„Radwechsel“ verwendet das vollständige Formular der Neueinlagerung mit allen
Pflichtfeldern, Kopierbutton für die Hinterachse ohne Profiltiefe, Zubehör,
Namensvorschlägen und direkter Platzprüfung. Die Zuordnung erfolgt serverseitig
über den eigenen Endpunkt `POST /api/wheel-changes`; dieselben JSON-Felder,
Validierungsregeln und Statuscodes wie `/api/storage`.

Der Vorgang wird in `storage_details.operation` als `wheel_change` gespeichert.
Die Antwort enthält `operation: "wheel_change"`. Die normale Neueinlagerung
speichert weiterhin `storage`. Die Such-API liefert zusätzlich `operation` und
kennzeichnet Radwechsel in der Oberfläche. Reservierungen behalten `reservation`.
Ein frei übermitteltes `operation`-Feld ist nicht erlaubt; der Endpunkt bestimmt
den Vorgangstyp.

Der aktuelle Ablauf entspricht wie gewünscht der Neueinlagerung: freier Platz
oder passende Reservierung mit Kunden-/Kennzeichenabgleich, atomare Speicherung,
Schließen nach Erfolg ohne Änderung der Suchleiste, leeres Formular beim Öffnen.
Ein bereits belegter Platz bleibt gesperrt. Der Austausch eines zuvor eingelagerten
Radsatzes inklusive Auslagerungsbuchung ist ein zusätzlicher, noch nicht
implementierter Vorgangsablauf.

Migration `003_storage_operation.sql` ergänzt ausschließlich die Vorgangskennung.
Bestehende Detaildatensätze erhalten `storage`. Vorhandene Daten bleiben erhalten;
keine neue Bibliothek. Server stoppen und einmalig mit automatischem Backup upgraden:

```sh
git switch beta
git pull --ff-only origin beta
go run . -dev -migrate
```

Für die normale Datenbank `-dev` weglassen. Danach weiterhin `go run . -dev` bzw.
`go run .`. Beim Test einen fiktiven Namen und freien Platz verwenden, beide
Achsen ausfüllen und speichern; anschließend nach Kennzeichen suchen und den
Hinweis „Radwechsel“ prüfen.

## Auslagerung

„Auslagerung“ öffnet ein eigenes Formular mit Lagerbereich/-nummer und direkter
Belegungsabfrage. Angezeigt werden Kunde, Kennzeichen, Fahrzeug und Lagerplatz.
Die Checkbox bestätigt den **angezeigten** Radsatz; erst danach ist der
Speicherbutton aktiv. Freie Plätze und Reservierungen können nicht ausgelagert
werden. Beim Schließen werden alle Felder geleert.

Nach Erfolg schließt das Formular. Eine vorhandene Schnellsuche wird erneut
abgefragt, ihr Suchtext bleibt unverändert. Die ausgelagerte Belegung bleibt als Historie auffindbar,
und der Platz kann trotzdem neu belegt oder reserviert werden.
Kunde, Fahrzeug, Radsatz, Detaildaten und ursprünglicher Einlagerungszeitstempel
bleiben erhalten. Es wird nichts gelöscht.

- `POST /api/releases/check`: JSON `{"area":"C1","number":25}`.
  Antwort enthält `place`, `status` (`free`, `reserved`, `occupied`), `message`;
  bei Belegung zusätzlich `claim_id`, `name`, `plate`, `car`.
- `POST /api/releases`: JSON
  `{"area":"C1","number":25,"claim_id":1,"confirmed":true}`.
  200 bestätigt die Auslagerung. 422 für fehlende Bestätigung/ungültigen Platz,
  409 für nicht mehr passende aktive Belegung, 400 für ungültiges JSON,
  403 fremder Ursprung, 415 falscher Medientyp, 500 Speicherfehler.
  Antworten verwenden `Cache-Control: no-store`.

Die finale Transaktion prüft die Kennung der zuvor angezeigten Belegung am
gewählten Platz. Eine inzwischen neue Belegung wird nicht freigegeben.
`storage_claims.status` wird auf `released` gesetzt; die neue Tabelle
`storage_releases` protokolliert Kennung und Auslagerungszeitpunkt. Beides ist
atomar. Ein Fehler erhält die Belegung; wiederholte/parallele Bestätigungen können
nicht eine weitere Auslagerung erzeugen. Der ursprüngliche Zeitstempel in
`storage_claims.updated_at` bleibt erhalten.

Migration `004_storage_releases.sql` ergänzt ausschließlich die Protokolltabelle.
Server stoppen, Code aktualisieren und mit Backup upgraden:

```sh
git switch beta
git pull --ff-only origin beta
go run . -dev -migrate
```

Danach genügt `go run . -dev`; für die normale Datenbank `-dev` weglassen.
Zum Test ausschließlich einen fiktiv belegten Platz wählen, angezeigte Daten
prüfen, Checkbox setzen und bestätigen. Anschließend denselben Platz in der
Neueinlagerung prüfen: Er muss frei sein. Eine Reservierung wird hier nicht
storniert; vollständige Historienansicht folgt später.

## Durchsuchbare Vorgangshistorie

Jede erfolgreich ausgeführte Neueinlagerung, jeder Radwechsel, jede Reservierung
und jede Auslagerung erzeugt **in derselben Transaktion** einen Historieneintrag.
Er enthält Vorgangsart, Zeitpunkt, Lagerplatz, Kundenname, Kennzeichen, Fahrzeug
und einen JSON-Schnappschuss der zugehörigen Detaildaten. Spätere Änderungen am
Fahrzeug verändern frühere Einträge nicht. Schlägt das Historienprotokoll fehl,
wird auch der eigentliche Vorgang vollständig zurückgerollt.

Die Schnellsuche durchsucht jetzt alle diese Ereignisse statt nur aktive
Belegungen. Suche nach Name, Kennzeichen oder Lagerplatz; auch die Vorgangsbegriffe
„Neueinlagerung“, „Radwechsel“, „Reservierung“ und „Auslagerung“ werden gefunden.
Neueste Ereignisse stehen oben. Mehrere Treffer für denselben Radsatz sind seine
unterschiedlichen Vorgänge. In der Oberfläche stehen Vorgangsart, aktueller
Lebenszykluszustand und Ereigniszeitpunkt. Eine Auslagerung ist als „Ausgelagert“
markiert und belegt **keinen** Platz; die Freiplatzprüfung verwendet weiterhin
nur aktive Belegungen/Reservierungen. Eine neue Belegung desselben Platzes kann
neben den früheren Historieneinträgen gefunden werden.

Nach jedem erfolgreichen Vorgang wird ein bereits eingegebener Suchtext erneut
abgefragt, ohne ihn zu verändern oder das zuletzt gespeicherte Kennzeichen
einzusetzen. Die separate Historie-Schaltfläche bleibt vorerst Platzhalter;
die Vorgangshistorie ist über die direkte Schnellsuche zugänglich.

`GET /api/search?q=...` behält das JSON-Array und die bisherigen Felder. Zusätzlich:
`event_id`, `status` (aktueller Zustand des zugehörigen Anspruchs), `active`
(aktuell wirksame Einlagerung/Reservierung) und `source` (`recorded`/`legacy`).
`operation` kann jetzt auch `release` enthalten, ebenso `kind: "release"`.
`date` ist der Zeitpunkt des **Ereignisses**. Leerzeichen-/Bindestrichnormalisierung
und Unicode-Suche bleiben erhalten; Ergebnisse bleiben `no-store`.

Migration `005_history_events.sql` ergänzt `history_events` und übernimmt
nachweisbare alte Vorgänge aus `storage_claims`, Details und `storage_releases`.
Diese Einträge sind als „Altbestand“ markiert. Bei früher umgewandelten
Reservierungen kann nur der bislang gespeicherte (möglicherweise bereits
aktualisierte) Zeitstempel übernommen werden; frühere Fahrzeugstände wurden
bisher ebenfalls nicht getrennt gespeichert. Nicht vorhandene frühere Angaben
werden nicht erfunden. Die Übernahme erfolgt genau einmal in der versionierten
Migration. SQL-Trigger verhindern nachträgliches Ändern/Löschen der Ereignisse.

Server stoppen und einmalig upgraden (mit Integritätsprüfung und automatischem,
konsistentem Backup vor der Migration):

```sh
git switch beta
git pull --ff-only origin beta
go run . -dev -migrate
```

Danach wieder `go run . -dev`; für die normale Datenbank jeweils `-dev` weglassen.
Zum Test einen fiktiven Radsatz einlagern und auslagern. Derselbe Suchtext muss
beide Ereignisse liefern, während die Platzprüfung den Platz als frei anzeigt.
Keine zusätzlichen Abhängigkeiten; Kundendaten und Datenbanken bleiben lokal.

## Automatische PDF-Belege (DIN A4)

Nach jeder erfolgreichen Neueinlagerung, jedem Radwechsel, jeder Reservierung und
jeder Auslagerung öffnet sich der jeweilige Beleg in einem neuen Tab. Dort über die
PDF-Werkzeugleiste oder Strg+P drucken. Bei ungültigen Eingaben oder einem
Speicherfehler wird kein Beleg geöffnet. Das Formular schließt weiterhin erst nach
Erfolg und der Suchtext bleibt unverändert.

Der Browser erhält den Tab unmittelbar beim Speicherklick; nach bestätigtem
Speichern lädt darin die echte PDF-Datei. Blockiert der Browser neue Tabs oder
wird der vorbereitete Tab geschlossen, enthält die Speicherbestätigung einen
„PDF öffnen“-Link. Bei Speicherfehlern wird der vorbereitete Tab geschlossen.
Die historische Suche bietet zusätzlich „Beleg als PDF“ zum erneuten Öffnen.
Eine Störung beim PDF-Abruf setzt keinen bereits gespeicherten Vorgang zurück;
Beleg erneut über den Link öffnen, den Vorgang nicht noch einmal speichern.

Die Belege enthalten die Angaben des unveränderlichen Historieneintrags:
Vorgangsart/-nummer, Ereigniszeitpunkt (Europe/Berlin), Lagerplatz, Kunde,
Kennzeichen und Fahrzeug. Bei Einlagerung, Radwechsel und Auslagerung außerdem
Reifen-/Felgenart, Größen VA/HA, Profiltiefen, Zubehör samt Aufbewahrungsorten,
Monteur und Bemerkungen, soweit erfasst. Reservierungsbelege enthalten nur die
dort erfassten Daten. Fehlende Altbestandsangaben stehen als „Nicht erfasst“ im
Beleg; historische Belege enthalten einen Altbestandshinweis.

Genau eine Seite DIN A4 im Hochformat, weißer druckfreundlicher Hintergrund.
Zeilen werden umgebrochen und die Schriftgröße vor der Ausgabe an den Umfang
angepasst; keine zusätzlichen Seiten und kein Abschneiden langer Bemerkungen.
Überflüssige Leerzeichen/Zeilenumbrüche werden für den Druck verdichtet. Die
DejaVu-Schrift wird mitgeliefert/eingebettet, damit Umlaute ohne lokale
Schriftinstallation funktionieren. Font-Lizenz: `internal/receipts/fonts/LICENSE.txt`.

Technik: neue pure-Go-Abhängigkeit `github.com/go-pdf/fpdf`; keine Cloud, kein
Browser/Office-Programm und keine weitere Systemabhängigkeit auf dem App-Server.
PDFs werden beim Abruf im Speicher aus der Historie erzeugt, nicht als zusätzliche
Kundendateien im Repository abgelegt. Kein Datenbankupgrade für dieses Feature.

Alle vier Speicherantworten und die Suchtreffer enthalten `pdf_url`.
`GET /api/receipts/{claim_id}/{operation}.pdf` liefert `application/pdf` mit
`Content-Disposition: inline` und `Cache-Control: private, no-store`.
Operationen: `storage`, `wheel_change`, `reservation`, `release`.
400 bei ungültigem Pfad, 404 bei unbekanntem Historieneintrag, 500 bei Abruffehler;
interne Datenbankdetails werden nicht ausgegeben. Der Beleg liest ausschließlich
den damaligen Schnappschuss und ändert keine Daten.

Update wie üblich mit gestopptem Server: `git pull --ff-only origin beta`,
anschließend `go run . -dev`. Falls das vorherige Historienupgrade noch fehlt,
einmalig zusätzlich `-migrate` wie oben beschrieben. Die tatsächliche Anzeige
und Druckauswahl hängen vom PDF-Verhalten des Browsers ab; für das automatische
Öffnen gegebenenfalls Popups/Tabs für die lokale App erlauben.

## Lizenz und Mitarbeit

Reifenapp ist unter der [MIT-Lizenz](LICENSE) verfügbar. Du darfst den eigenen
Projektcode verwenden, verändern und weitergeben, auch kommerziell. Dabei müssen
der Urhebervermerk und Lizenztext erhalten bleiben. Die Software wird ohne
Gewährleistung bereitgestellt.

Bibliotheken und Schriftdateien behalten ihre eigenen Lizenzen; siehe
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). Die Projektlizenz bezieht sich
auf den veröffentlichten Projektinhalt. Lokale Kunden- und Fahrzeugdaten,
Datenbanken, Backups und Zugangsdaten bleiben ausschließlich lokal.

Fehlerberichte und Verbesserungsvorschläge sind über
[GitHub Issues](https://github.com/karst3nl/reifenapp-2/issues) willkommen.
Beiträge bitte gegen `beta` richten, den bestehenden Code und `AGENTS.md` beachten
und passende Tests ausführen. Für Beispiele, Tests, Screenshots und Fehlerberichte
nur fiktive Daten verwenden. Keine echten Kundendaten, Datenbankdateien, Backups
oder Zugangsdaten in Issues, Pull Requests oder Commits veröffentlichen.

## Drucklayout nach der bereitgestellten HTML-Vorlage

Die PDFs übernehmen den Aufbau der früheren Druckansicht: großer Firmenkopf mit
Adresse/Telefon, zweispaltige Kunden-/Fahrzeugdaten mit Trennlinien sowie
Reifen-/Felgenart/Felgenschloss, Radkappen/Profiltiefen und Radschrauben/Hinweise
in kompakten Spalten. Neue Angaben wie Reifengrößen VA/HA, Monteur, Vorgangsart
und Historienreferenz bleiben enthalten. Datum und Beschriftungen passen zur
Vorgangsart; Reservierungen enthalten weiterhin nur ihre erfassten Felder.

Der Firmenkopf ist lokal konfigurierbar. Bei Bedarf die ignorierte Datei
`local/receipt-header.json` im Editor anlegen (Beispiel vollständig fiktiv):

```json
{
  "company": "Werkstatt Muster GmbH (fiktiv)",
  "street": "Musterstraße 1",
  "city": "00000 Musterstadt",
  "phone": "0000/12345"
}
```

Die eigenen Firmenangaben einsetzen und den Server neu starten. Ohne diese Datei
wird der neutrale Kopf „Reifenapp 2.0“ verwendet. Leere optionale Zeilen werden
weggelassen; maximal 100 Zeichen pro Konfigurationszeile. Abweichender Pfad:
`go run . -dev -receipt-config /pfad/zum/firmenkopf.json`.
Eine fehlerhafte Datei wird beim Start gemeldet; sie wird nicht überschrieben.
Firmenkonfiguration nicht in Git committen. Die hochgeladene Originaldatei und
ihre realen Firmendaten werden nicht ins öffentliche Repository übernommen.

Automatische PDFs, neuer Tab, Strg+P, Ersatzlink bei Popup-Blockierung und
Nachdruck aus der Historie funktionieren weiterhin. Genau eine Seite DIN A4:
bei ungewöhnlich langen Werten reduziert sich zunächst die Schriftgröße;
falls nötig folgt eine kompakte Tabellenanordnung mit demselben Firmenkopf,
damit alle Daten erhalten bleiben. Keine weitere Migration oder Abhängigkeit.
Die zusätzliche fette DejaVu-Schrift ist unter derselben bestehenden Fontlizenz
mitgeliefert. Die Startseite und die Formulare werden nicht neu gestaltet.

## Administration: frei gestaltbarer DIN-A4-PDF-Designer

Über „Administration“ öffnet sich ein eigener passwortgeschützter Bereich mit
DIN-A4-Arbeitsfläche. Text, Datenfelder, Linien und Flächen frei ziehen, skalieren,
färben und anordnen. Millimeterwerte, Schriftgröße/Fettdruck, Ausrichtung,
Duplizieren, Ebenen und Rückgängig/Wiederholen stehen bereit. Eigene Vorlage pro
Vorgangsart, echte PDF-Vorschau und lokale Speicherung in SQLite.

**Einrichtung:** Nach `go run . -dev -migrate` unter Administration mit Benutzer
`admin` und Startpasswort `Reifenapp-Start2026!` anmelden. Direkt ein eigenes
Passwort festlegen und wiederholen; erst danach wird der Designer freigeschaltet.
Das eigene Passwort bleibt als bcrypt-Hash lokal gespeichert. Keine
Umgebungsvariable mehr nötig. Normale Vorgänge bleiben ohne Login; automatische
PDFs und Ausdruck auf einer A4-Seite bleiben erhalten.

[Anleitung, Passwort-Einrichtung und API](docs/pdf-designer.md).
Bilder/Logos sind in der ersten Version noch nicht enthalten. Passwort-Hashing erfolgt mit `golang.org/x/crypto/bcrypt`.

### Raster im PDF-Designer

Ein sichtbares Raster hilft beim Anordnen: 5 mm als Standard, außerdem 1/2/10 mm
oder „Aus“. Elemente rasten beim Ziehen und Skalieren ein; auch neue Elemente,
Duplikate und die Pfeiltasten orientieren sich daran. Alt erlaubt vorübergehend
freie Bewegung, und manuelle Millimeterwerte bleiben präzise einstellbar.
Das Raster skaliert mit der Ansicht, verändert vorhandene Vorlagen beim Umschalten
nicht und erscheint nicht im PDF. Kein Datenbankupgrade erforderlich.
