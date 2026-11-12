# Admin-Panel und PDF-Designer

Der Designer ist der erste Administrationsbereich. Normale Werkstattvorgänge
benötigen weiterhin keine Anmeldung. Die Startseite behält ihr bestehendes Design.
Auf „Administration“ klicken, um den geschützten Bereich zu öffnen.

## Einrichten und aktualisieren

Server vor dem Update stoppen. In Bash (das Passwort wird nicht angezeigt und
steht nicht als Klartext im eingegebenen Shellbefehl):

```sh
git switch beta
git pull --ff-only origin beta
read -rs -p 'Admin-Passwort: ' REIFENAPP_ADMIN_PASSWORD
printf '\n'
export REIFENAPP_ADMIN_PASSWORD
go run . -dev -migrate
```

Ein eigenes Passwort mit 12 bis 128 Bytes verwenden. Es gibt kein Standardpasswort.
Der Server liest es beim Start aus der Umgebung; es wird nicht in SQLite oder
Browser-Storage gespeichert und nicht protokolliert. Für die normale Datenbank
`-dev` weglassen. Der Prozess muss die Umgebungsvariable bei jedem Start erhalten.
Ohne sie bleibt die Administration deaktiviert; die Werkstattfunktionen laufen.
Bei einem Neustart werden bestehende Admin-Sitzungen ungültig.

Migration `006_pdf_layouts.sql` ergänzt nur die lokale Vorlagentabelle. Das
explizite Upgrade erstellt vorher wie bisher ein konsistentes SQLite-Backup nach
Integritätsprüfung. Vorhandene Kunden, Radsätze und Historieneinträge bleiben
unverändert. Nach dem Upgrade genügt wieder `go run . -dev` bzw. `go run .`.

## Gestalten

- Vorgangsart wählen: Neueinlagerung, Radwechsel, Reservierung oder Auslagerung.
  Jede hat eine eigene Vorlage.
- Auf der weißen DIN-A4-Seite (210 × 297 mm) vorhandene Elemente auswählen und
  ziehen. Der Griff unten rechts ändert Breite und Höhe.
- Text, Datenfelder, horizontale Linien und Rechtecke/Flächen hinzufügen.
  Datenfelder erhalten später die echten Werte des jeweiligen Historieneintrags.
- Position und Abmessungen rechts in Millimetern einstellen. Schriftgröße,
  Fettdruck, Ausrichtung, Text-/Linienfarbe und Flächenfarbe bearbeiten.
- Duplizieren, löschen und nach vorn/hinten legen. Überlagerungen sind möglich,
  etwa ein Rahmen hinter einem Text. „Rückgängig“/„Wiederholen“ bzw. Strg+Z/Y.
- Pfeiltasten verschieben das ausgewählte Element um 0,5 mm, mit Umschalt um 2 mm.
- Ansicht 50/75/100 % beeinflusst nur die Bildschirmdarstellung, nicht das Papier.

Die Arbeitsfläche nutzt eine lokale eingebettete Schrift und fiktive Beispieldaten.
Die genaue Ausgabe prüft „PDF-Vorschau“: Sie verwendet denselben PDF-Generator und
Entwurf wie der spätere Beleg. Auch noch nicht gespeicherte Änderungen sind sichtbar.
Es werden keine echten Kundendatensätze für die Vorschau geladen.

„Ausgangsvorlage laden“ ersetzt den Entwurf durch eine bearbeitbare Startvorlage.
Es wird noch nichts aktiviert. „Speichern & aktivieren“ speichert den Entwurf und
verwendet ihn sofort für die gewählte Vorgangsart. „Standardbeleg verwenden“
schaltet zurück auf die bisherige automatische Druckgestaltung; der eigene Entwurf
bleibt gespeichert. Die Revision schützt vor Überschreiben einer zwischenzeitlich
in einer anderen Sitzung gespeicherten Vorlage. Bei Konflikt neu laden, nicht
blind erneut speichern. Eine vollständige Liste früherer Vorlagenversionen ist
nicht Teil dieses ersten Designers; Rückgängig/Wiederholen gilt für die Sitzung.

## PDF-Ausgabe

Automatische neue Tabs nach Werkstattvorgängen, Druck per Strg+P und Nachdrucklinks
bleiben erhalten. Eine Layoutänderung wirkt auch auf später erneut abgerufene
Belege aus der Historie; deren fachliche Daten bleiben der unveränderte damalige
Schnappschuss. Die Vorlage wird innerhalb der gewählten lokalen Datenbank gespeichert
(Entwicklungs- und normale Datenbank sind getrennt), nicht in GitHub.

Alle Elemente müssen innerhalb A4 liegen; maximal 100 Elemente, je höchstens 2000
Zeichen statischer Text. Erlaubte Datenfelder sind fest definiert; kein HTML,
JavaScript, externe Ressourcen oder Dateipfade in Vorlagen. Bilder/Logos und
mehrseitige Dokumente sind in dieser ersten Version noch nicht vorgesehen.
Bewusst entfernte Datenfelder erscheinen im eigenen Layout nicht mehr.

Text wird umgebrochen und bei Bedarf innerhalb des Feldes bis 6 pt verkleinert.
Ist ein Feld weiterhin zu klein, lehnt die Vorschau die Ausgabe mit einer
verständlichen Meldung ab. Bei einem echten, unerwartet langen Kundendatensatz
wird stattdessen ein vollständiger einseitiger Standardbeleg erzeugt. Dieser
enthält einen sichtbaren Hinweis zum Layout-Fallback. Der Vorgang bleibt gespeichert;
Daten werden nicht abgeschnitten und eine zweite A4-Seite wird nicht angelegt.

## Anmeldung und API

Sitzung: 30 Minuten, zufälliges Token im HttpOnly-/SameSite-Strict-Cookie, Secure
bei direktem HTTPS. Admin-Schreibzugriffe benötigen zusätzlich ein sitzungsgebundenes
CSRF-Token und gleichherkünftiges JSON. Anmeldeversuche werden pro Client begrenzt.
Das Umgebungs-Passwort wird zum Vergleich in einem konstantzeitgeprüften Hash im
Prozess gehalten; es gibt keine dauerhafte Passwortdatenbank. Die Umgebung und
lokale Startkonfiguration entsprechend privat halten. Für Netzwerkbetrieb den
Transport-/Zugriffsschutz beim Bereitstellen berücksichtigen; Standardstart bleibt
auf 127.0.0.1 beschränkt.

Admin-Endpunkte:

- `POST /api/admin/login` mit `{ "password": "…" }`, danach Session-Cookie und
  CSRF-Wert aus der Antwort benutzen.
- `GET /api/admin/session`; `POST /api/admin/logout`.
- `GET /api/admin/layout?operation=storage` liefert gespeicherte Vorlage und
  Ausgangsvorlage. Andere Operationen: `wheel_change`, `reservation`, `release`.
- `POST /api/admin/layout` mit `operation`, `template`, `enabled`, `revision`.
- `POST /api/admin/preview` mit derselben Struktur liefert eine echte A4-PDF.
- `GET /api/admin/fields`, `/api/admin/values?operation=…` und `/api/admin/font`
  liefern Feldkatalog, fiktive Vorschauwerte und die Schrift.

Schreibzugriffe außer Login benötigen `X-CSRF-Token`. 401: nicht angemeldet,
403: Sitzung/Ursprung ungültig, 409: Revisionskonflikt, 422: Layout ungültig,
429: Anmeldebegrenzung, 503: Administration nicht eingerichtet. Layoutdaten und
PDFs werden nicht öffentlich zwischengespeichert.
