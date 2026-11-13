# Betrieb und HTTP-API

Diese Anleitung beschreibt den aktuellen Entwicklungsstand auf `beta`.
Für Installation und Schnellstart siehe [README](../README.md).

## Lokale Daten und Migrationen

`go run .` verwendet `data/reifenapp.sqlite`, `go run . -dev` die getrennte
`data/development.sqlite`. Nur eine neu erstellte Entwicklungsdatenbank erhält
fiktive Beispieldaten. Vorhandene Datenbanken werden nicht neu befüllt.

SQLite verwendet WAL, Fremdschlüssel und einen Busy-Timeout. Versionierte
SQL-Migrationen liegen in `migrations/`. Eine neue Datenbank wird initialisiert;
ein bestehendes Schema wird beim Start geprüft. Erforderliche Upgrades müssen
explizit mit `-migrate` gestartet werden:

```sh
go run . -migrate
# Oder ausschließlich für Entwicklungsdaten:
go run . -dev -migrate
```

Vorher alle anderen Prozesse mit Zugriff auf die jeweilige Datenbank beenden.
Das Upgrade prüft die Integrität und erstellt mit `VACUUM INTO` ein konsistentes
Backup einschließlich noch im WAL befindlicher Änderungen. Die Sicherung liegt
neben der ursprünglichen Datei als `*.sqlite.backup-<UTC-Zeitstempel>`.
Bei einem Backupfehler findet keine Migration statt. Danach laufen die
Migrationen transaktional. Keine ältere App-Version auf dem aktualisierten
Schema starten und Datenbankdateien nicht als vermeintliche Fehlerbehebung löschen.

Daten, Sicherungen und lokale Konfigurationen sind ignoriert und gehören nicht
in Git. Für eine eigene Datensicherung ein konsistentes SQLite-Backup verwenden;
bei laufendem WAL-Betrieb nicht lediglich die einzelne Hauptdatei kopieren.

## Lagerregeln und Historie

Die folgenden Bereiche sind die Ausgangskonfiguration. Lagerorte und Nummernbereiche
sind über die [Lagerplatzverwaltung](lagerplaetze.md) im Adminbereich anpassbar.

| Bereich | Nummern |
| --- | --- |
| C1 | 1–200 |
| C2 | 201–400 |
| C3, C4, OL, OR | jeweils 1–60 |

Kennungen bestehen aus Bereich und Nummer, etwa `C1.2`. Aktive Belegungen und
Reservierungen verwenden gemeinsame Lageransprüche (`storage_claims`). Ein
partieller UNIQUE-Index verhindert mehrere aktive Ansprüche auf denselben Platz;
auch ein Radsatz darf nur einen aktiven Anspruch haben.

Eine Neueinlagerung darf eine Reservierung nur übernehmen, wenn Kundenname und
Kennzeichen beide übereinstimmen. Der Namensabgleich ignoriert Großschreibung und
verdichtet Leerzeichen; beim Kennzeichen werden zusätzlich Bindestriche ignoriert.
Eine nicht zugeordnete Altreservierung wird nicht automatisch übernommen.

Bei einer Auslagerung wird die aktive Belegung atomar freigegeben. Der Vorgang
bleibt als unveränderlicher Schnappschuss in der Suche erhalten. Nachträglich aus
Altbeständen erzeugte Historieneinträge können nur noch vorhandene Informationen
abbilden, keine bereits überschriebenen Werte rekonstruieren.

## HTTP-API

Die Werkstatt-Endpunkte benötigen derzeit keinen Login. Der Server bindet
standardmäßig an `127.0.0.1:8080`. Die Tabelle beschreibt die Routen als Überblick;
konkrete JSON-Strukturen und Validierungen stehen in `internal/handlers/` und
`internal/models/`.

| Methode und Pfad | Zweck |
| --- | --- |
| `GET /api/warnings?scope=active` | Verschleißwarnungen für aktuell eingelagerte Radsätze; `scope=all` umfasst alle gespeicherten Vorgänge |
| `GET /api/season` | Saison und Wechselstand des aktuellen Bestands; siehe [Saisonanzeige](saison.md) |
| `GET /api/history` | Vollständige Historie mit Filtern und Seitenwechsel; siehe [Historienanleitung](historie.md) |
| `GET /api/search?q=...` | Suche in der Vorgangshistorie nach Kunde, Kennzeichen, Platz oder Vorgangsart |
| `GET /api/customers?q=...` | Bis zu 20 Namensvorschläge per Präfixsuche |
| `GET /api/storage/options` | Auswahlwerte und aktueller Lagerkatalog (`area`, `storage_areas`) |
| `POST /api/storage/check` | Lagerplatz für Einlagerung oder Reservierung prüfen |
| `POST /api/storage` | Neueinlagerung speichern |
| `POST /api/wheel-changes` | Einlagerung mit Vorgangsart Radwechsel speichern |
| `POST /api/reservations` | Freien Platz reservieren |
| `POST /api/releases/check` | Aktuelle Belegung für eine Auslagerung ermitteln |
| `POST /api/releases` | Belegung nach ausdrücklicher Bestätigung freigeben |
| `GET /api/receipts/{claim_id}/{operation}.pdf` | PDF eines gespeicherten Vorgangs abrufen |

### Suche

Eine leere Suchanfrage liefert `[]`; mehr als 200 Zeichen werden mit HTTP 400
abgewiesen. Die Suche ignoriert Groß-/Kleinschreibung einschließlich Umlauten
sowie Leerzeichen und Bindestriche. Ergebnisse enthalten unter anderem `place`,
`name`, `plate`, `car`, `date`, `operation`, `status`, `active` und `pdf_url`.
Abgeschlossene Vorgänge bleiben auffindbar, auch wenn der Platz wieder frei ist.

`details` enthält die damaligen Angaben zu Marke/Modell, Reifen-/Felgenart,
Reifengrößen und Profiltiefen beider Achsen, Zubehör samt Aufbewahrungsorten,
Monteur und Hinweisen. Die Daten stammen aus dem unveränderlichen Vorgangsschnappschuss,
nicht aus einer späteren Belegung desselben Platzes. Bei Reservierungen und
Altbeständen werden nicht erfasste Angaben nicht ergänzt.

Auf der Startseite lassen sich die Ergebniskarten per Pfeil oder mit der Tastatur
aufklappen. Hinweise erscheinen bereits in der geschlossenen Karte gelb/orange
mit Warnsymbol. Für die Profiltiefe gelten die festgelegten App-Warnschwellen:
**Sommer ≤ 3 mm**, **Winter/Ganzjahr ≤ 4 mm**, jeweils getrennt für VA und HA.
Die Karte nennt Messwert und Warnschwelle. Fehlende Messwerte oder unbekannte
Reifenarten lösen keine erfundene Verschleißbewertung aus. Historische Warnungen
beziehen sich auf den Messwert des jeweiligen Vorgangs; sie sind keine neue Messung.


### Schreiben und Platzprüfung

JSON-Anfragen verwenden `area` und `number` für den Platz, `name` und `plate`
für die Zuordnung. Einlagerung und Radwechsel benötigen außerdem Marke, Modell,
Reifen-/Felgenart sowie vollständige Reifendaten und Profiltiefen beider Achsen.
Profiltiefen sind von 0 bis 10 mm in Schritten von 0,5 mm zulässig.
Zubehör ist optional; als vorhanden markiertes Zubehör benötigt einen Lagerort.
Reservierungen benötigen nur Platz, Kunde, Kennzeichen, Marke und Modell.

`/api/storage/check` unterscheidet über `purpose` zwischen `storage` und
`reservation` und meldet freie, belegte oder reservierte Plätze. Eine Vorprüfung
ersetzt nicht die transaktionale Kontrolle beim Speichern. Im Radwechselformular
entfällt die laufende Vorabprüfung einschließlich „Erneut prüfen“. Die bestehenden
serverseitigen Regeln gegen doppelte Belegung und fremde Reservierungen gelten
beim Speichern weiterhin; ein automatischer Austausch belegter Radsätze ist
damit nicht verbunden.
Bei einer Auslagerung müssen `claim_id` der zuvor ermittelten Belegung und
`confirmed: true` mitgesendet werden. So wird keine inzwischen geänderte Belegung
versehentlich freigegeben. Reservierungen werden darüber nicht storniert.

## PDF-Belege und Firmenkopf

Erfolgreiche Vorgänge liefern `pdf_url`. Die Oberfläche öffnet den Beleg in einem
neuen Tab; blockiert der Browser Popups, steht ein Ersatzlink bereit. Ausdruck
über die PDF-Ansicht beziehungsweise Strg+P. Nachdruck ist über Suchtreffer möglich.

Operationen heißen `storage`, `wheel_change`, `reservation` und `release`.
PDFs entstehen lokal im Speicher aus dem gespeicherten Historienschnappschuss,
werden mit eingebetteten Schriften auf einer A4-Seite ausgegeben und nicht als
zusätzliche Kundendateien im Repository gespeichert. Abrufe verwenden
`Content-Disposition: inline` und `Cache-Control: private, no-store`.
Die aktive Gestaltungsvorlage gilt auch für Nachdrucke; die Vorgangsdaten bleiben
unverändert. Bei überlaufenden eigenen Layouts dient das Standardlayout als Rückfall.

Der Firmenkopf des Standardlayouts kann über die ignorierte Datei
`local/receipt-header.json` konfiguriert werden. Beispiel ausschließlich mit
fiktiven Angaben:

```json
{
  "company": "Werkstatt Muster GmbH (fiktiv)",
  "street": "Musterstraße 1",
  "city": "00000 Musterstadt",
  "phone": "0000/12345"
}
```

Danach Server neu starten. Ohne Datei wird der neutrale Kopf „Reifenapp 2.0“
verwendet. Maximal 100 Zeichen je Zeile; ein abweichender Pfad ist mit
`-receipt-config /pfad/zur/datei.json` möglich. Fehlerhafte Dateien werden gemeldet,
nicht überschrieben. Details zu eigenen Layouts und Passwörtern stehen in der
[Designer-Anleitung](pdf-designer.md).

## LAN und Raspberry Pi

`-addr 0.0.0.0:8080` erlaubt das Lauschen auf allen Netzwerkschnittstellen.
Vor LAN-Betrieb Netz, Firewall und Zugriffsberechtigungen festlegen: Der Admin-Login
schützt nicht die übrigen Werkstatt-Endpunkte, und der Server stellt selbst kein
TLS bereit. Für den lokalen Standardstart ist diese Option nicht erforderlich.

Auf dem Raspberry Pi werden Go und eine C-Toolchain benötigt. Zunächst direkt
auf dem Zielgerät bauen; Cross-Compiling mit CGO benötigt eine passende
Cross-Toolchain. Ein Hardwaretest auf dem Pi steht noch aus.

## Warnungsanzeige auf der Startseite

Oben rechts öffnet **Reifenwarnungen** eine Übersicht unabhängig von der Suche.
Der Zähler zählt aktuell eingelagerte Radsätze mit mindestens einer betroffenen
Achse, nicht die Anzahl der Achsen. Zwei kritische Achsen desselben Radsatzes
zählen deshalb als ein Radsatz. Die Übersicht nennt zusätzlich die Achsenzahl.

**Aktuell eingelagert** zeigt aktive Einlagerungen und Radwechsel. Nach einer
Auslagerung entfällt der Vorgang im aktuellen Zähler. **Alle Vorgänge inkl.
Historie** zeigt sämtliche gespeicherten Vorgangsschnappschüsse mit kritischen
Messwerten, auch frühere Einlagerungen und Auslagerungen. Dort können mehrere
Vorgänge desselben Radsatzes erscheinen; sie sind mit Vorgangsart, Status und
Zeitpunkt gekennzeichnet. Eine erneute gesunde Belegung desselben Platzes übernimmt
keine alten Warnungen. Alle Karten lassen sich aufklappen und bieten den PDF-Beleg.

Die Warnschwellen bleiben Sommer ≤ 3 mm und Winter/Ganzjahr ≤ 4 mm. Suchkarten und
Warnungsübersicht verwenden dieselbe serverseitige Auswertung. Fehlende oder
ungültige Messwerte und unbekannte Reifenarten erzeugen keine Bewertung.
Ein bloßer Freitexthinweis wird nicht als Reifenverschleiß gezählt.

Der Zähler lädt beim Seitenstart, nach erfolgreichen Formularvorgängen, bei
Rückkehr zum Browserfenster und auf sichtbaren Seiten jede Minute neu. Öffnen und
„Aktualisieren“ laden die Übersicht erneut. Bei Abruffehlern erscheint `?` statt
einer falschen Null; die Übersicht bietet eine Fehlermeldung und erneutes Laden.
Es werden keine neuen Warnungsdatensätze erzeugt und keine Kundendaten verändert.

`GET /api/warnings` verwendet standardmäßig `scope=active`. Die JSON-Antwort enthält
`scope`, `count` (Radsätze bzw. historische Vorgänge), `axle_count` und `items`.
Jeder Eintrag enthält die bekannten Suchergebnisfelder sowie `tread_warnings`
mit `axle` (`VA`/`HA`), `depth` und `threshold`. `scope=all` liefert alle passenden
Historienvorgänge ohne stillschweigende Kürzung. Ungültige Ansichten liefern HTTP 400,
Datenbankfehler HTTP 500. Antworten verwenden `Cache-Control: no-store`.
Für dieses Feature ist keine weitere Datenbankmigration nötig.
