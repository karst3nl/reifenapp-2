# Lagerorte und Nummernbereiche verwalten

Unter **Administration → Lagerplätze** lassen sich neue Lagerorte anlegen und
Nummernbereiche bestehender Orte ändern. Der **PDF-Editor** ist ein eigener Bereich
in derselben Administration. Beim Wechsel bleibt ein ungespeicherter PDF-Entwurf
erhalten; vor dem Verwerfen offener Lagerplatzänderungen erscheint eine Nachfrage.

## Vorhandene Installation aktualisieren

Server und alle anderen Prozesse mit Zugriff auf die Datenbank stoppen.
Nach dem Aktualisieren von `beta` einmal mit `-migrate` starten:

```sh
go run . -migrate
# Für die getrennte Entwicklungsdatenbank stattdessen:
go run . -dev -migrate
```

Migration `008_storage_areas.sql` übernimmt die bisherigen sechs Bereiche und alle
Platzkennungen. Vor dem Upgrade erfolgen Integritätsprüfung und konsistentes Backup.
Kunden, Belegungen, Reservierungen, Historie, Admin-Passwort und PDF-Vorlagen bleiben
erhalten. Anschließend reicht der normale Start ohne `-migrate`.

## Lagerort anlegen

1. Mit dem eigenen Admin-Passwort anmelden und „Lagerplätze“ öffnen.
2. „Neuen Lagerort anlegen“ wählen.
3. Kennung, erste und letzte Lagernummer eingeben und speichern.

Beispiel: `HALLE2`, erste Nummer `100`, letzte Nummer `150` erzeugt 51 auswählbare
Plätze von `HALLE2.100` bis `HALLE2.150`. Der Bereich ist einschließlich beider Grenzen.

Kennungen dürfen 1–12 Großbuchstaben, Ziffern, Unterstriche oder Bindestriche
enthalten. Nummern sind ganzzahlig von 1 bis 100000; pro Ort sind höchstens 10000
zusammenhängende Plätze möglich. Gleiche Nummern in unterschiedlichen Lagerorten
sind zulässig, vollständige Platzkennungen sind eindeutig.

## Bestehenden Bereich bearbeiten

Die Übersicht zeigt den Zahlenbereich und die Anzahl aller, belegten, reservierten
und freien Plätze. Über „Bearbeiten“ lassen sich erste und letzte Nummer ändern.
Die Vorschau zeigt den resultierenden Bereich vor dem Speichern.

- **Erweitern:** Zusätzliche Nummern stehen anschließend zur Verfügung.
- **Verkleinern oder verschieben:** Nur möglich, wenn alle wegfallenden Plätze
  weder aktiv belegt noch aktiv reserviert sind. Andernfalls wird die gesamte
  Änderung abgewiesen; es findet keine automatische Auslagerung oder Stornierung statt.
- **Historie:** Wegfallende Platzkennungen werden deaktiviert, nicht gelöscht.
  Frühere Vorgänge bleiben suchbar und ihre Belege abrufbar. Wird der Bereich
  später erweitert, können die alten Nummern wieder aktiviert werden.
- **Gleichzeitige Bearbeitung:** Veraltete Änderungen werden mit einem Konflikt
  abgewiesen. „Neu laden“ wählen und den aktuellen Bereich erneut bearbeiten.

Bestehende Lagerortkennungen werden nicht umbenannt. Eine automatische Umnummerierung
belegter Radsätze und das vollständige Löschen von Lagerorten sind nicht Bestandteil
dieses Bereichs. So bleiben die bisherigen Zuordnungen erhalten.

Neueinlagerung, Reservierung, Radwechsel und Auslagerung laden die aktuelle
Auswahl beim Öffnen neu. Bereits offene Formulare werden beim Speichern zusätzlich
gegen den aktuellen Katalog geprüft. Nach einer Bereichsänderung ein offenes
Formular schließen und erneut öffnen, um seine Auswahlliste zu aktualisieren.

## Technik und API

`storage_areas` enthält Kennung, Nummerngrenzen und Bearbeitungsrevision.
`storage_places.enabled` bestimmt, welche Kennungen aktuell auswählbar sind.
Historische Plätze und ihre Fremdschlüssel bleiben bestehen. Bereichsänderungen
und Platzaktualisierung laufen gemeinsam in einer unmittelbaren SQLite-Transaktion.
Aktive Ansprüche auf deaktivierte Plätze sind zusätzlich durch Trigger gesperrt;
die bisherigen UNIQUE-Indizes gegen doppelte Belegung bleiben erhalten.

Die folgenden Endpunkte benötigen eine Admin-Sitzung nach abgeschlossenem
Passwortwechsel; Schreibzugriffe benötigen zusätzlich `X-CSRF-Token`:

- `GET /api/admin/areas`: Bereiche inklusive Belegungszahlen und Revision laden.
- `POST /api/admin/areas`: Bereich anlegen oder ändern, etwa
  `{ "code": "HALLE2", "first_number": 100, "last_number": 150, "revision": 0 }`.
  Revision `0` bedeutet Neuanlage; Änderungen benötigen die geladene Revision.
  Erfolg liefert die aktualisierte Liste, ungültige Eingaben HTTP 422,
  Bearbeitungs- und Belegungskonflikte HTTP 409.

`GET /api/storage/options` liefert für die Werkstattformulare weiterhin die Liste
`area` und zusätzlich `storage_areas` mit den aktuellen Nummerngrenzen.

## Fehlende Lagerbewegung

Unter **Administration → Lagerplätze** lässt sich die Meldefrist auf 1 bis 120
Monate einstellen (Standard: 12 Monate). Die Einstellung wird lokal in SQLite
gespeichert und ist im vollständigen Datenbankbackup enthalten.

Oben rechts auf der Startseite zeigt **Ohne Bewegung** die Anzahl betroffener
aktiver Belegungen. Ein Klick öffnet die Übersicht mit Kunde, Kennzeichen,
letzter erfasster Bewegung und vergangenen Tagen. Die Liste wird bei Vorgängen,
beim Zurückkehren zur Seite und jede Minute aktualisiert.

Als Bewegung zählt die letzte tatsächlich erfasste Einlagerung oder der letzte
Radwechsel der aktuellen Belegung. Die Monatsfrist wird ab diesem Zeitpunkt in
der Zeitzone Europe/Berlin berechnet (Kalendermonate mit Go AddDate). Ein erneuter
Radwechsel startet die Frist neu; eine Auslagerung entfernt die Meldung.
Reservierungen, freie Plätze, CSV-Archive und nachträglich rekonstruierte
Altbestandszeitpunkte gelten nicht als Bewegungsnachweis. Fehlende, ungültige
oder zukünftige Zeitpunkte werden als unklar gezählt, ohne eine Liegezeit zu erfinden.

Für das Update ist Migration 010 nötig: Server und andere Datenbankzugriffe
beenden, dann `go run . -migrate` starten. Vor der Migration wird automatisch
eine konsistente Sicherung erstellt.
