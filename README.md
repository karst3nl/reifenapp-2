# Reifenapp 2.0

Lokale Reifenlagerverwaltung mit Go, SQLite und HTML/CSS/Vanilla JavaScript.
Die Original-README enthielt ausschließlich den Repository-Namen `reifenapp-2`.

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
- Ladezustand, keine Treffer und Fehleranzeige; Neueinlagerung ist nutzbar, die übrigen fünf Vorgänge bleiben Platzhalter.
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
Einziger direkter externer Go-Baustein ist der bewährte SQLite-Treiber
`github.com/mattn/go-sqlite3`. SQLite verwendet WAL, Fremdschlüssel und einen
Busy-Timeout. SQL-Migrationen laufen transaktional bei der Erstinitialisierung.
Bestehende Datenbanken werden auf die bekannte Schemaversion geprüft; unbekannte
Versionen werden abgelehnt. Schema 1 benötigt das unten beschriebene explizite
Upgrade mit Integritätsprüfung und konsistentem SQLite-Backup.

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
Einmalig nötig bei Schema 1: Integritätsprüfung, dann `VACUUM INTO` für eine
konsistente Sicherung inklusive noch im WAL befindlicher Änderungen, danach
transaktionale Migration 002. Das Backup liegt neben der ursprünglichen Datei
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
   aktive Reservierung sperrt den Platz; keine automatische Reservierungsumwandlung.

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
Start weiterhin `go run . -dev` (ein Schema-1-Upgrade benötigt einmalig zusätzlich
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
