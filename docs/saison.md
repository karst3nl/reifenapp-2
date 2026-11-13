# Saisonanzeige und Wechselstand

Die Startseite zeigt links neben „Reifenwarnungen“ die automatisch ermittelte
Saison und den Stand **gewechselt / zugeordnet** für den aktuellen Lagerbestand.
Die Berechnung liest vorhandene Daten, verändert keine Vorgänge und benötigt keine
zusätzliche Datenbankmigration.

## Kalender

Als Voreinstellung gelten:

- **Wintersaison:** 1. Oktober bis einschließlich 31. März, etwa „Winter 2026/27“.
- **Sommersaison:** 1. April bis einschließlich 30. September, etwa „Sommer 2026“.

Maßgeblich ist die Zeitzone **Europe/Berlin** für die Werkstatt bei Mainz, unabhängig
von der Zeitzone des Browsers oder der Betriebssystemeinstellung des Servers.
Die Zeitzonendaten werden mit dem Go-Programm ausgeliefert. Das funktioniert auch
auf minimalen Linux-/Raspberry-Pi-Installationen ohne zusätzliches tzdata-Paket.

## Was der Zähler zählt

Gezählt werden aktive **Belegungen**, je ein Radsatz pro Belegung. Reservierungen
und bereits ausgelagerte Radsätze gehören nicht zum aktuellen Bestand.

| Einordnung | Wintersaison | Sommersaison |
| --- | --- | --- |
| Gewechselt | Gebuchter Radwechsel dieser Saison mit eingelagerten Sommerreifen | Gebuchter Radwechsel dieser Saison mit eingelagerten Winterreifen |
| Noch offen | Winterreifen noch eingelagert | Sommerreifen noch eingelagert |
| Separat | Ganzjahresreifen sowie nicht eindeutig zuordenbare Bestände | Ganzjahresreifen sowie nicht eindeutig zuordenbare Bestände |

**Nenner = gewechselt + offen.** Zwei kritische Profiltiefen haben auf diese Zählung
keinen Einfluss; Verschleißwarnungen bleiben ein eigener Bereich.

Beispiel in der Wintersaison: 12 aktuelle Sommerreifen-Einlagerungen mit passendem
Radwechselbeleg und 8 noch eingelagerte Winterradsätze ergeben **12 / 20 gewechselt**.
3 Ganzjahressätze und 2 unklare Bestände werden daneben genannt, nicht in die 20
hineingerechnet. Mehrere Radsätze desselben Kunden bleiben mehrere Radsätze.

Die Einordnung „gewechselt“ benötigt einen als `wheel_change` gespeicherten Vorgang
mit Zeitpunkt innerhalb der aktuellen Saison und nicht in der Zukunft. Eine reine
Neueinlagerung oder eine Reservierung ist kein Wechselbeleg. Das Reifenprofil
beschreibt dabei die **eingelagerten**, also beim Wechsel abgenommenen Reifen.

Fehlende Reifenart, nicht belegbare Altbestände und Einlagerungen der erwarteten
Reifenart ohne aktuellen Radwechselbeleg erscheinen als **unklar**. Erfasste
Vorgangsschnappschüsse bleiben die Datengrundlage; fehlende historische Angaben
werden nicht nachträglich aus veränderlichen Details ergänzt.

Beim Radwechsel werden die Radsätze auf dem bisherigen Kundenplatz in einer
Transaktion getauscht. Der alte Satz wird aus der aktiven Belegung genommen, der
neue Satz eingelagert. Eine vorherige manuelle Auslagerung ist dafür nicht nötig.
So wird beispielsweise aus einem offenen Winterradsatz ein erledigter Wechsel mit
Sommerreifen im Lager, während der zugeordnete Gesamtbestand gleich bleibt.
Voraussetzung ist die Übereinstimmung von Kundenname und Kennzeichen mit der
bisherigen Belegung. Bekannte saisonale Sätze werden bei späteren Wechseln
wiederverwendet; alte Profiltiefen und Vorgangsdaten bleiben in der Historie.

Die Anzeige ist ein Fortschritt des **jetzigen Bestands**, keine unveränderliche
Gesamtzahl aller Saisontermine: Wird ein Radsatz dauerhaft ausgelagert, entfällt
er im Zähler. Die zugehörigen Vorgänge bleiben in der Historie erhalten.

## Aktualisierung und Fehler

Die Anzeige lädt beim Seitenstart, nach erfolgreichen Formularvorgängen, bei
Rückkehr zum Browserfenster und auf sichtbaren Seiten jede Minute neu. Dadurch
wechselt auch die Kalenderanzeige automatisch in die nächste Saison.
Bei Ladefehlern steht „Nicht verfügbar“ statt eines irreführenden Nullstands.
Ein leerer zugeordneter Bestand erscheint als „0 / 0 gewechselt“; vorhandene unklare
oder Ganzjahresbestände bleiben daneben sichtbar. Die Erläuterung ist zusätzlich
als Tooltip und für Bildschirmleser verfügbar.

## API

`GET /api/season` liefert die aktuelle Saison und folgende Zahlen:

- `season`: `kind`, `label`, `mounted_type`, `stored_type`, `start`, `end`.
- `completed`: eindeutig zugeordnete Radwechsel im aktuellen Bestand.
- `open`: noch eingelagerte Radsätze der auf dem Fahrzeug erwarteten Reifenart.
- `total`: `completed + open`.
- `unknown`: nicht eindeutig zugeordnete Bestände.
- `all_season`: Ganzjahressätze.
- `active`: alle aktiven Belegungen, also Summe der vier Kategorien.

Die Zeitgrenzen enthalten die Mainzer Ortszeit inklusive Zeitversatz. Der Start
ist inklusiv, das Ende exklusiv. Die Serveruhr bestimmt den aktuellen Zeitpunkt.
Antworten verwenden `Cache-Control: no-store`; Datenbankfehler liefern HTTP 500.
Schreibzugriffe und ein künstliches Datum über URL-Parameter sind nicht vorgesehen.
