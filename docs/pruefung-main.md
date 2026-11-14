# Abnahme vor Übernahme nach main

**Datum:** 14. November 2026. **Umgebung:** Linux amd64, Go 1.25.4,
System-Chromium/Playwright. Ausschließlich temporäre Datenbanken und fiktive Daten.

Geprüft wurde der Stand nach den Zubehör-Pflichtfeldern und der Raspberry-Pi-
Betriebsanleitung. Die Übernahme nach `main` erfolgt auf ausdrücklichen Nutzerauftrag;
weitere Entwicklung bleibt auf `beta`. Dieser Bericht bestätigt den unten genannten
Prüfumfang, nicht die Vollständigkeit aller ursprünglich geplanten Ausbaustufen.

## Ergebnis und Abgrenzung

**Die Anforderungen des ursprünglichen ersten Meilensteins sind erfüllt.** Die
später ausdrücklich beauftragten Werkstattabläufe sind implementiert. Die ursprüngliche
Projektübersicht enthält darüber hinaus eine Roadmap. Nicht alle dort genannten
Ausbauziele sind bereits vollständig umgesetzt; siehe „Offene Punkte“.

Frühere Vorgaben wurden durch spätere Nutzerentscheidungen angepasst:

- Das neue Repository ist inzwischen ausdrücklich öffentlich und MIT-lizenziert.
- Helleres Design, geänderte Startseitenanordnung und sanfte Übergänge sind beauftragte
  Weiterentwicklungen der ursprünglichen Vorlage, keine unbegründete Neugestaltung.
- Die Monteureingabe wurde ausdrücklich aus den Formularen entfernt. Frühere gespeicherte
  Angaben bleiben erhalten; neue Monteurstatistiken sind kein aktueller Auftrag.
- Der Radwechsel tauscht saisonale Radsätze desselben Kunden/Fahrzeugs am selben Platz.
  Er erzeugt keine gesonderte Auslagerung. Ein Platz hat höchstens eine aktive Belegung.
- Anfangs geplante Platzhalter wurden durch echte Formulare, Historie und Administration ersetzt.

## Abgleich der ursprünglichen Anforderungen

| Anforderung | Ergebnis / Nachweis |
| --- | --- |
| Ausschließlich neues Repo ReifenApp-2 verwenden | `https://github.com/karst3nl/reifenapp-2.git`; aktueller GitHub-Zugriff und Schreibrechte bestätigt. Das alte Reifensuch-App-Repo wurde nicht verwendet oder verändert. |
| Bestehende Dateien vorher lesen und erhalten | Bestehende Historie erhalten. Drei ältere `main`-Dokumentationscommits eingebunden; aktuelle README, Screenshot und vollständige Lizenzhinweise aus `beta` nach Prüfung beibehalten. Kein Force-Push. |
| Go mit wenigen Abhängigkeiten | Standardbibliothek für HTTP; drei direkte Module für SQLite, PDF und Passwort-Hashing. Kein Frontend-Framework oder Frontend-Build. |
| SQLite, WAL, versionierte Migrationen | Zwölf SQL-Migrationen, Foreign Keys, WAL, Transaktionen, automatisches Backup vor Schema-Upgrades. Automatisierte Tests und Startprüfung erfolgreich. |
| `go run .` aus dem Repo-Stamm | In frischer isolierter Projektkopie gestartet; leere Produktionsdatenbank, HTTP-Gesundheitsprüfung und kontrolliertes Beenden per SIGTERM geprüft. |
| Suchfeld direkt mit Go/SQLite verbinden | Name, Kennzeichen und Lagerplatz über `GET /api/search`; keine fest eingebettete JS-Kundendatenbank. Tests für leere Treffer, Fehler, Unicode und SQL-Sonderzeichen. |
| Sechs Hauptschaltflächen | Radwechsel, Neueinlagerung, Auslagerung, Reservierung, Historie, Administration im Browser vorhanden und angebunden. |
| Ausgangsbereiche und Nummern | C1 1–200, C2 201–400; C3/C4/OL/OR je 1–60; insgesamt 640 Plätze. Weitere Bereiche/Nummern konfigurierbar. |
| Höchstens eine aktive Belegung, keine Reservierungskollision | Gemeinsamer partieller UNIQUE-Index für aktive Platzansprüche, atomare Reservierungsübernahme und konkurrierende Schreibtests. |
| Fiktive Entwicklungs-/Testdaten getrennt halten | Normale Datenbank startet leer; `-dev` nutzt eigene Datei, automatische Tests temporäre Datenbanken. Keine echte Kundendatenbank getestet oder veröffentlicht. |
| Keine lokalen Daten/Backups/Geheimnisse/Builddateien versionieren | Dateinamen des aktuellen Baums und der Git-Historie auf Datenbank-/CSV-/Backupdateien geprüft; keine solchen Kundendateien gefunden. Erkennung verbreiteter Token-/Private-Key-Muster ohne Treffer. `.gitignore` zusätzlich für timestampbasierte SQLite-Kopien und Standard-Binärdateien ergänzt; keine Garantie einer umfassenden externen Geheimnisprüfung. |
| Migrationen, Code und geeignete fiktive Testdaten versionieren | Migrationen und Go-Testfixtures versioniert; SQLite-Dateien und echte Alt-CSV bleiben außerhalb des Repos. |
| Kleine nachvollziehbare Commits | Entwicklung schrittweise auf `beta`; ursprüngliches `main` ist im Übernahmeverlauf enthalten. |
| Tag v0.1.0 erst nach erstem Meilenstein | Vorhandener Tag `v0.1.0` und damaliger Prüfbericht geprüft; Tag wird weder verschoben noch ersetzt. |
| AGENTS.md und verständliche Struktur | Arbeitsregeln im Stamm; `internal`, `web`, `migrations`, `docs`, später `deploy`; Tests neben dem Go-Code. |
| CachyOS und Raspberry Pi dokumentieren | README mit CachyOS-Start; SSH-Anleitung mit systemd, Datenübernahme, LAN-Adresse, GitHub-Updates, Backup, Rückfall und USB-CUPS. Tatsächliche Pi-Hardwareprüfung bleibt offen. |
| Architektur und Start kurz dokumentieren | README, Betriebs-/API-Anleitung und Fachanleitungen vorhanden; produktive Installations- und Updateanleitungen verweisen jetzt auf `main`. |

## Später beauftragte Funktionen

Implementiert und durch Backendtests sowie ausgewählte Browserabläufe abgesichert:

- Neueinlagerung, Radwechsel, Reservierungsübernahme und Auslagerung mit fortbestehender
  Historie; Kundennamenvorschläge, Pflichtfelder und unmittelbare Lagerplatzprüfung.
- Vorder-/Hinterachse mit Pflichtdaten, getrennten Profiltiefen 0–10 mm in 0,5-mm-Schritten,
  Übernahme der VA-Reifendaten ohne Übernahme der HA-Profiltiefe.
- Pflichtauswahl vorhanden/nicht vorhanden und anpassbare Aufbewahrungsort-Dropdowns für
  Zubehör; Prüfung aktueller Werte in der Schreibtransaktion. Alte Angaben bleiben erhalten.
- Suchkarten mit Details, Hinweisen und Verschleißwarnungen: Sommer ≤ 3 mm,
  Winter/Ganzjahr ≤ 4 mm. Historie mit Filtern, Seitenwechsel und Verläufen.
- Saisonfortschritt, reservierte/freie Plätze und konfigurierbare Meldungen bei fehlender Bewegung.
- Einseitige A4-Belege, PDF-Editor mit Raster/Mittellinie und Logoimport.
- Alt-CSV mit Vorschau, Archivkennzeichnung, Originalwerterhaltung und Dublettenschutz;
  Archivimporte verändern den aktuellen Lagerbestand nicht.
- Vollständige Datenbankbackups einschließlich Historie, Einstellungen, Vorlagen und Importen;
  lesbare HTML-/JSON-Zugaben. Wiederherstellung in einer separaten temporären Datenbank geprüft.
- Etikettenvorlagen 78 × 51 mm, persistente automatische Aufträge nach Neueinlagerung/Radwechsel,
  lokaler USB-CUPS-Zugriff, Testdruck, Status und kontrollierter neuer Versuch. Die physische
  Ausgabe am ARGOX wurde nicht geprüft; CUPS-Übergaben wurden mit simulierten Befehlen getestet.
- Kontrolliertes Beenden, Migration ohne HTTP-/Druckstart und Versions-/Datenbankprüfung
  unter `/api/health` für den systemd-/SSH-Betrieb.

## Aktuell ausgeführte Prüfungen

- `go test -race ./...`, `go vet ./...`, Build und `git diff --check`: erfolgreich.
- JavaScript-Syntaxprüfungen und `bash -n deploy/update.sh`: erfolgreich.
- Echte Browserformulare: Neueinlagerung/Radwechsel, Zubehörpflichtfelder und eigene
  Adminlisten; aktuelle Liste nach einer zwischenzeitlichen Adminänderung übernommen,
  sonstige Formulardaten beibehalten. Reservierung ohne unpassende Zubehörpflichtfelder.
- Browser: CSV-Vorschau/Import, Historienfilter/Originalwerte, A4- und Etikettenvorschau,
  Backend-Backup als ZIP, Warnungen inklusive Archiv, Zähler und 390px-Ansicht.
  Keine JavaScript-Laufzeitfehler in diesen Abläufen.
- Backup nach den Testvorgängen separat geöffnet: SQLite-Integrität `ok`, Schema 12,
  native und importierte Historie sowie Zubehörlisten vorhanden.
- Start aus frischem Projektstamm mit `go run .`: keine Entwicklungsdaten in der
  Produktionsdatenbank; WAL und HTTP geprüft. SIGTERM endet kontrolliert.
- Update-Skript mit Standardbranch `main` gegen isoliertes lokales Git-Remote:
  nativer Build, Dienstwechsel, konsistentes Backup, Migration und exakte Commit-
  Prüfung über reales HTTP erfolgreich. `sudo`, systemd und SQLite-CLI dabei über
  Testadapter simuliert; SQLite-Backup-API, Go-Binärdatei und HTTP waren real.
  Die Update-Fehlerpfade wurden zusätzlich beim SSH-Deployment-Auftrag mit fiktiven
  Daten geprüft (Backup-/Migrationsfehler, abweichender Checkout, falsche Health-Antwort).
- systemd-Datei zuvor statisch mit angepasstem Prüf-Binärpfad validiert. Kein echter
  systemd-Systemdienst in dieser Cloud-Umgebung gestartet.
- GitHub-Repository öffentlich, Standardbranch `main`, Schreibrechte vorhanden;
  vor der Übernahme keine offenen Pull Requests gefunden.

Der README-Screenshot wurde aus dem aktuellen Stand mit ausschließlich fiktiven
Abnahmedaten neu aufgenommen.

## Offene Punkte und nicht ausgeführte Prüfungen

Die folgenden Punkte sind **keine vollständig erledigten Roadmap-Ziele**:

1. **Weitere Admin-Dropdowns:** Lagerorte und Zubehörorte sind konfigurierbar.
   Automarken, Reifen-/Felgenarten, Reifengrößen und Profiltiefen sind weiterhin
   feste Auswahllisten. Eine allgemeine Verwaltung aller Auswahlkategorien fehlt.
2. **Konsolidierte Kunden-/Fahrzeugprofile:** Suche und Historie bieten Kunden-,
   Fahrzeug- und Platzverläufe. Eine eigenständige Stammdaten-/Profilpflege mit
   konsolidierter Zuordnung mehrerer Fahrzeuge/Radsätze ist nicht implementiert.
3. **Reservierungs-Stornierung/Ablaufverwaltung:** Das Schema kennt weitere Status;
   aktuell sind Reservierung und passende Umwandlung implementiert, keine vollständige
   Verwaltungsoberfläche für Stornierungen oder automatisch auslaufende Reservierungen.
4. **Hardware-/Betriebsabnahme:** Kein Test auf echtem CachyOS-Rechner oder Raspberry Pi,
   kein realer Pi-Neustart, kein Werkstatt-LAN-/SSH-Mehrgerätetest und kein physischer
   ARGOX-Ausdruck. Die dokumentierten Schritte müssen am Zielsystem geprüft werden.

Der Softwarestand kann mit diesen transparenten Grenzen nach `main` übernommen werden.
Eine Zusicherung, dass sämtliche ursprünglichen Zukunftsziele oder die reale Hardware
vollständig abgenommen seien, wäre durch diese Prüfung nicht gedeckt.
