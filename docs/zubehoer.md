# Zubehör: Pflichtauswahl und anpassbare Dropdowns

Bei **Neueinlagerung** und **Radwechsel** muss für jedes Zubehör ausdrücklich
**Vorhanden** oder **Nicht vorhanden** ausgewählt werden:

- Radkappen
- Radschrauben
- Felgenschloss

Eine leere Auswahl ist keine Verneinung. Ist Zubehör vorhanden, ist zusätzlich
sein Aufbewahrungsort Pflicht. Bei „Nicht vorhanden“ wird der Ort geleert und
ist nicht auswählbar. Es wird nicht verlangt, dass das Zubehör tatsächlich vorhanden ist.
Reservierungen und Auslagerungen erhalten keine zusätzlichen Pflichtfelder.

## Listen bearbeiten

**Administration → Zubehör** öffnen. Für jede Zubehörart gibt es eine eigene Liste:
Ein Aufbewahrungsort pro Zeile, in der gewünschten Reihenfolge. 1–50 eindeutige
Einträge je Zubehör, jeweils höchstens 200 Zeichen ohne Steuerzeichen. Leere Zeilen
werden in der Oberfläche ignoriert. Doppelte Einträge, auch bei unterschiedlicher
Groß-/Kleinschreibung, sind nicht zulässig. Die letzte Auswahlmöglichkeit einer Liste
kann nicht entfernt werden.

Nach dem Speichern laden neu geöffnete Formulare die aktuellen Dropdowns. War ein
Formular bereits geöffnet und wurde dessen ausgewählter Ort inzwischen entfernt,
weist die API den Vorgang mit einer Feldmeldung ab und die Oberfläche lädt die
aktuellen Orte nach. Die übrigen Formulardaten bleiben erhalten.

Die Bedeutungen „Vorhanden“/„Nicht vorhanden“ sind fest vorgegeben; bearbeitbar sind
die Aufbewahrungsorte. Weitere Zubehörarten werden durch die Listen nicht angelegt.

## Bestehende Daten

Änderungen an den Listen benennen gespeicherte Vorgänge nicht um. Alte Angaben
bleiben in Historie, Suche, Backups und Belegen erhalten, auch wenn ein damaliger
Aufbewahrungsort inzwischen aus der Liste entfernt wurde. CSV-Archivdaten werden
nicht nachträglich auf neue Pflichtangaben umgeschrieben.

Die Listen liegen versioniert mit Revisionskontrolle in SQLite und sind Teil des
vollständigen Datenbankbackups. Gleichzeitige Adminänderungen werden nicht still
überschrieben. Für den Update auf diese Funktion ist Migration 012 erforderlich:
Server stoppen, Code aktualisieren und `go run . -migrate` starten. Beim Pi-Dienst
stattdessen das [SSH-Update-Skript](raspberry-pi.md#5-updates-aus-github-per-ssh) verwenden.

## API

- `GET /api/storage/options`: Aktuelle Orte unter `hubcaps.location`, `bolts.location`,
  `lock.location` sowie die festen Präsenzwerte unter `accessory_presence`.
- `GET/POST /api/admin/accessory-options`: Admin-Login und bei POST CSRF-Schutz;
  JSON mit `revision` und `locations` für alle drei Zubehörarten.
- Neue Anfragen an `POST /api/storage` und `POST /api/wheel-changes` müssen jedes
  Zubehör als Objekt mit explizitem booleschem `present` enthalten, z. B.
  `"hubcaps": {"present": false, "location": ""}`. Fehlend oder `null` ist nicht zulässig.
- Bei `present: true` wird `location` innerhalb derselben Datenbanktransaktion gegen
  die aktuelle Liste geprüft. Fehlende oder ungültige Angaben ergeben HTTP 422 mit
  Feldmeldungen; es entstehen weder Belegung noch Historien- oder Druckauftrag.
