# Historie

Die Schaltfläche **Historie** auf der Startseite öffnet `/history.html`.
Die Seite liest die vorhandenen Vorgangsschnappschüsse aus SQLite. Sie benötigt
keine zusätzliche Migration und verändert oder löscht keine Kundendaten.

## Übersicht und Suche

Alle gespeicherten Neueinlagerungen, Radwechsel, Reservierungen und Auslagerungen
erscheinen mit den neuesten Vorgängen zuerst. Die vier Zähler beziehen sich auf
alle Treffer des aktuellen Filters, nicht nur auf die sichtbare Seite.

Die Suche berücksichtigt Kundenname, Kennzeichen, Lagerplatz, Fahrzeug, Vorgangsart,
Hinweise und Monteur. Groß-/Kleinschreibung, Leerzeichen und Bindestriche werden
ignoriert; Umlaute werden unterstützt. Weitere Filter:

- Vorgangsart und aktueller Lagerstatus;
- Zeitraum von/bis einschließlich des gewählten Tages, in der Zeitzone des Browsers;
- erfasste Vorgänge oder übernommenen Altbestand;
- Vorgänge mit niedriger Profiltiefe nach den bestehenden Warnschwellen.

„Filter zurücksetzen“ zeigt wieder alle Vorgänge. Mit 20, 50 oder 100 Einträgen
pro Seite sind auch ältere Vorgänge ohne Suchbegriff erreichbar.

## Vorgangsdetails und Verläufe

Die Karten zeigen Lagerplatz, Name, Kennzeichen und Fahrzeug in einer gemeinsamen
Kopfzeile. Per Pfeil oder Tastatur lassen sich Reifengrößen, Profiltiefen, Zubehör
und dessen Lagerorte, Monteur und Hinweise aufklappen. „Beleg als PDF“ öffnet den
Vorgang zum Nachdrucken über die vorhandene PDF-Erzeugung.

„Kundenverlauf“, „Fahrzeugverlauf“ und „Platzverlauf“ setzen die übrigen Filter zurück
und suchen nach dem jeweiligen Namen, Kennzeichen beziehungsweise Lagerplatz.
Das sind Textsuchen, keine automatische Zusammenführung von Kunden mit gleichem Namen.

Die fachlichen Angaben bleiben die damaligen Werte. Beispielsweise behält eine
Reservierung ihr ursprüngliches Fahrzeug, auch wenn bei der späteren Einlagerung
andere Angaben erfasst wurden. Der angezeigte **Lagerstatus heute** bezieht sich
dagegen auf die aktuelle Belegung oder Reservierung dieses Vorgangs. Ein älterer
Vorgang wird nicht wieder aktiv, wenn derselbe Platz neu belegt wird.

Bei Altbeständen sind nicht mehr vorhandene Angaben als nicht erfasst gekennzeichnet.
Sie werden nicht erfunden. Historische Verschleißwarnungen beziehen sich auf die
damalige Messung. PDFs verwenden die aktuell aktivierte Druckvorlage und die
unveränderten damaligen Vorgangsdaten.

## Radwechsel auf dem bisherigen Kundenplatz

Ein Radwechsel protokolliert die Entnahme des bisher eingelagerten Satzes als
Auslagerung und die Einlagerung des anderen Satzes als Radwechsel mit demselben
Lagerplatz und Zeitpunkt. Beide Bewegungen werden atomar gespeichert. Das alte
Reifenprofil und der alte Beleg bleiben erhalten; nur die neue Belegung ist aktiv.
Bei späteren Wechseln kann der bereits bekannte Sommer- beziehungsweise Wintersatz
wieder demselben Platz zugeordnet werden.

## Aktualisieren und Fehlerfälle

Eine Ereignis-ID-Grenze hält neu hinzugekommene Vorgänge während des Blätterns aus
der bestehenden Seitenauswahl heraus. „Aktualisieren“ oder eine Filteränderung
startet eine neue Auswahl und nimmt neue Vorgänge auf. Änderungen des heutigen
Lagerstatus können weiterhin die Treffer eines Statusfilters verändern.

Ladefehler, keine Treffer und ungültige Datumsbereiche werden ausdrücklich angezeigt.
Mit „Aktualisieren“ lässt sich erneut laden. Es gibt hier keine Bearbeitung,
Löschung oder rückwirkende Änderung von Vorgängen.

## HTTP-API

`GET /api/history` liefert `items`, `total`, `page`, `page_size`, `pages`, `snapshot`
und `operations` (Treffer je Vorgangsart). Leere Auswahl: `items: []`, `total: 0`,
`pages: 0`. Antworten verwenden `Cache-Control: no-store`.

| Parameter | Bedeutung |
| --- | --- |
| `q` | Suchtext, höchstens 200 Zeichen |
| `operation` | `storage`, `wheel_change`, `reservation` oder `release` |
| `status` | `active` oder `closed`; aktueller Lagerstatus des Vorgangs |
| `source` | `recorded` oder `legacy` |
| `from` | Inklusiver Startzeitpunkt als RFC3339, einschließlich Zeitzone |
| `until` | Exklusiver Endzeitpunkt als RFC3339; Oberfläche sendet den Beginn des Folgetags |
| `warnings` | `true` für Vorgänge mit Verschleißwarnung |
| `page` | Seite ab 1; Standard 1 |
| `page_size` | 20, 50 oder 100; Standard 20 |
| `snapshot` | Ereignis-ID-Grenze aus der ersten Antwort für weitere Seiten; 0/fehlend startet neu |

Ungültige Parameter liefern HTTP 400, Datenbankfehler HTTP 500. Schreibmethoden
auf diesem Endpunkt sind nicht vorgesehen. Die Datenbank wird zeilenweise gelesen;
nur die angeforderte Seite wird im Ergebnis gehalten. Unicode-Suche und präzise
Zeitfilter werden in Go ausgewertet.
