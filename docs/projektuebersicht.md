# Reifenapp 2.0 – Projektübersicht für Codex

**Stand:** 10. November 2026  
**Projekt:** Reifenapp 2.0 – interne Reifen- und Radsatzlagerverwaltung einer Kfz-Werkstatt  
**GitHub-Repository:** `ReifenApp-2` (**privat**, exakt dieser Name). Das alte Reifensuch-App-Repository darf weder gelesen als Codebasis übernommen noch verändert werden.

> Diese Übersicht wurde aus den bisherigen Projektabsprachen für den neuen Codex-Chat rekonstruiert. Sie dokumentiert Zielanforderungen, nicht die Bestätigung, dass Funktionen bereits implementiert sind. Den tatsächlichen Repository-Stand vor Änderungen prüfen.

## 1. Ziele und Arbeitsweise

- Lokal betreibbare Webanwendung zur Lagerplatz-, Kunden-, Fahrzeug- und Radsatzverwaltung.
- Zugriff im lokalen Netzwerk über einen Webbrowser; zuerst Betrieb auf CachyOS, perspektivisch Raspberry Pi.
- Schnelle, übersichtliche Werkstattoberfläche mit möglichst wenigen Klicks und guter Suchfunktion.
- Keine Cloudpflicht für den App-Betrieb, keine unnötigen Frameworks, einfache Wartung.
- Quellcode in privatem GitHub-Repository versionieren; Änderungen lokal per `git pull` übernehmen. Die produktive SQLite-Datenbank, Kundendaten, lokale Einstellungen und Sicherungen bleiben ausschließlich lokal.
- Codex übernimmt Programmierung, kleine nachvollziehbare Schritte und Tests. Grundsatzentscheidungen werden vor größeren Architekturänderungen abgestimmt.

## 2. Technologie

- **Backend:** Go, bevorzugt Go-Standardbibliothek (`net/http`) für Webserver/API.
- **Persistenz:** SQLite im WAL-Modus mit klaren SQL-Migrationen und referenzieller Integrität; eine bewährte, notwendige Go-SQLite-Bibliothek ist zulässig.
- **Frontend:** HTML, CSS und Vanilla JavaScript, ohne JS-Framework und ohne Build-Pipeline soweit möglich.
- **Entwicklungsumgebung:** CachyOS/Linux, Editor Zed; perspektivisch Raspberry Pi unter Linux.
- Start soll einfach dokumentiert sein, idealerweise `go run .` aus dem Repository-Stamm.

## 3. Startseite / Design – verbindlich

Die bereits erstellte **Originaldatei `index.html`** muss zusätzlich bereitgestellt und als Vorlage übernommen werden. **Farben, Abstände, Icons, Layout und Beschriftungen zunächst unverändert lassen.** Die Datei ist derzeit eine HTML-Vorschau mit fünf fiktiven Datensätzen im JS-Array und Platzhalterdialogen – der produktive Code soll die Suche über Go/SQLite bedienen und die sechs Funktionen schrittweise umsetzen.

Startseite mit **direkter Schnellsuche** nach Kundennamen, Kennzeichen oder Lagerplatz, ohne eigenen Such-Knopf. Sechs sichtbare Hauptschaltflächen:

1. Radwechsel
2. Neueinlagerung
3. Auslagerung
4. Reservierung
5. Historie
6. Administration

Bei der ersten Etappe dürfen die sechs Schaltflächen noch Platzhalter sein; die Suche muss allerdings mit der Datenbank verbunden werden.

## 4. Lagerbereiche und Nummerierung

Sechs Lagerbereiche: **C1, C2, C3, C4, OL (Oben Links), OR (Oben Rechts)**.

| Bereich | Zulässige Nummern | Beispiel |
| --- | --- | --- |
| C1 | 1–200 | `C1.2` |
| C2 | 201–400 | `C2.215` |
| C3 | 1–60 | `C3.8` |
| C4 | 1–60 | `C4.1` |
| OL | 1–60 | `OL.12` |
| OR | 1–60 | `OR.4` |

- Schreibweise **`Bereich.Nummer`** (beispielsweise `C1.2`).
- Pro Lagerplatz höchstens **ein aktiv eingelagerter Radsatz**.
- Aktive Reservierungen dürfen weder mit aktiver Belegung noch mit anderen aktiven Reservierungen unzulässig kollidieren.
- Belegungen, Reservierungen und Bewegungen transaktionssicher behandeln und mit Constraints/geeigneten Prüfungen schützen, nicht nur im Frontend.

## 5. Fachliche Daten / Vorgänge

Ein Vorgang bzw. Radsatz soll nach Bedarf die folgenden Angaben erfassen:

- **Vorgangsart:** Radwechsel, Neueinlagerung, Auslagerung, Reservierung.
- **Lagerplatz**, Kunde (Name), Kennzeichen.
- Fahrzeug: Marke und Modell.
- Reifenart, Felgenart.
- Reifengröße **Vorderachse und Hinterachse**, jeweils Breite, Querschnitt/Höhe, Zoll, Traglast und Geschwindigkeitsindex.
- Profiltiefe **VA/HA** mit 0,5-mm-Schritten.
- Zubehör: Radkappen, Radschrauben, Felgenschloss; je Vorhandensein und Aufbewahrungsort.
- Monteur, Bemerkungen/Notizen, Zeitstempel.

Bei Reservierungen können Teilangaben statt vollständiger Einlagerungsdaten nötig sein; genaue Pflichtfeldregeln je Vorgangsart vor Implementierung bestätigen. Vorher keine unnötigen Formulare erzwingen.

## 6. Kundenprofil und Historie

- Ein Kunde kann mehrere Fahrzeuge und mehrere Radsätze/Lagerplätze haben.
- Kunden-/Fahrzeugübersicht mit aktuellem Lagerstatus und Historie der Vorgänge.
- Letzte Einlagerung sowie Hinweis auf verschlissene Reifen anhand der Profiltiefe darstellen; konkrete Verschleißschwellen vor Automatisierung festlegen.
- Änderungen an Belegungen und Reservierungen nachvollziehbar protokollieren.

## 7. Reservierungen und Lagerlogik

- Status einer Reservierung: `active`, `converted`, `cancelled`, `expired`.
- Umwandlung einer aktiven Reservierung in reale Einlagerung muss atomar erfolgen.
- Doppelbelegungen und unzulässige Doppelreservierungen zuverlässig verhindern.
- Beispiele für automatisierte Tests:
  - `C1.2` ist belegt → Reservierung für `C1.2` wird abgelehnt.
  - `C1.3` ist frei, aber reserviert → Umwandlung der passenden Reservierung in Belegung funktioniert.
  - Gleichzeitige Zugriffe dürfen nicht zwei aktive Belegungen desselben Platzes erzeugen.

## 8. Administration

- Für normale Werkstattvorgänge zunächst **kein allgemeiner Nutzerlogin** (schnelle Bedienung).
- **Admin-Login** für Änderungen an Verwaltungseinstellungen. Sicherheitsmaßnahmen und lokaler Netzzugriff müssen berücksichtigt werden.
- Admin pflegt Dropdown-Optionen, unter anderem Lagerbereich (ohne Nummer), Automarke, Reifenart, Felgenart, Reifengrößen- und Profiltiefen-Optionen.
- CSV-Import für Altbestand und lokale Backup-Möglichkeit sind geplant.
- Keine echten Daten oder Backups in GitHub-Commits, Pull Requests, CI-Logs oder Codex-Ausgaben veröffentlichen.

## 9. Datenmodell – fachliche Entitäten als Ausgangspunkt

Voraussichtlich benötigt: `customers`, `vehicles`, `tire_sets`, `storage_numbering_groups`, `storage_areas`, `storage_places`, `storage_occupancies`, `reservations`, `events`, `event_accessories`, `mechanics`, `dropdown_categories`, `dropdown_options`.

Das ist eine **Zielmodell-Skizze**, nicht die Anweisung, sofort alle Tabellen ohne Prüfung anzulegen. Erst das Minimum für die erste Suchfunktion implementieren; spätere Migrationen entwickeln, sobald die konkreten Vorgangsabläufe feststehen.

## 10. Weitere geplante Funktionen

- PDF-Zusammenfassung eines Vorgangs im A4-Format.
- CSV-Import des Altbestands.
- Backup-/Wiederherstellungsablauf für die SQLite-Datenbank.
- Historienansicht, Profiltiefenauswertung und Admin-Dashboard.

Diese Funktionen **nicht** ungefragt schon in Meilenstein 1 implementieren.

## 11. Verzeichnisstruktur – schlanker Vorschlag

```text
ReifenApp-2/
├── main.go
├── go.mod
├── go.sum                  # bei externen Modulen
├── internal/
│   ├── database/
│   ├── handlers/
│   └── models/
├── web/
│   ├── index.html         # Designvorlage beibehalten
│   ├── css/               # bei Bedarf
│   └── js/                # bei Bedarf
├── migrations/
├── docs/
│   └── projektuebersicht.md
├── data/                   # nur lokal; DB/Backups ignorieren
├── AGENTS.md
├── .gitignore
└── README.md
```

Tests möglichst idiomatisch als `*_test.go` neben dem Go-Code, eigener `tests/`-Ordner nur bei echtem Mehrwert. Leere Ordner und Platzhalterdateien vermeiden.

## 12. Dauerhafte Anweisungen für Codex / `AGENTS.md`

- Nur am Repository `ReifenApp-2` arbeiten, die alte Reifensuch-App unangetastet lassen.
- Bestehendes Design nicht ohne Rücksprache ändern.
- Keine produktiven Daten, SQLite-Dateien einschließlich `-wal`/`-shm`, Geheimnisse oder Backups committen.
- Klarer, kompakter Go-Code, unnötige Abhängigkeiten vermeiden.
- Vor Dateiänderungen Repository-Stand prüfen; mit kleinen Commits/PRs arbeiten und Tests durchführen.
- Keine bestehenden Datenbanken zurücksetzen oder migrieren, ohne Sicherheitsprüfung und Backup.
- README laufend für Start/Setup aktualisieren; Fehlversuche und nicht ausgeführte Tests offen nennen.

## 13. Erster Meilenstein (v0.1.0 erst nach Testfreigabe)

1. Neuen privaten Repository-Stand und Rechte prüfen; nichts mit dem alten Repository vermischen.
2. Projektgrundstruktur, `.gitignore`, README, `AGENTS.md` aufbauen.
3. Go-Webserver so einrichten, dass die Original-Startseite ausgeliefert wird.
4. SQLite initialisieren (inkl. WAL), notwendige erste Migration(en) anlegen.
5. API `GET /api/search` für Suche nach Name, Kennzeichen oder Platz erstellen (Parameternamen und Response in README dokumentieren).
6. JavaScript-Suchlogik auf HTTP-Abfrage umstellen, fiktives Demo-Array aus der produktiven Suchlogik entfernen. Ergebnisdarstellung und übrige Optik beibehalten.
7. Ausschließlich fiktive Testdaten und Tests einsetzen. Datenbank- und API-Tests für Suche, leere Ergebnisse und wichtige Fehlerfälle.
8. Build/Tests, Start mit `go run .` soweit möglich prüfen; Commits/PR und Report liefern.

**Wichtig:** Die Startseiten-Datei ist separate Designreferenz, nicht durch diese Übersicht ersetzbar. Bei fehlendem Zugriff zuerst um die echte Datei bitten.
