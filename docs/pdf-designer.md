# Admin-Panel und PDF-Designer

Der Designer ist der erste Administrationsbereich. Normale Werkstattvorgänge
benötigen weiterhin keine Anmeldung. Die Startseite behält ihr bestehendes Design.
Auf „Administration“ klicken, um den geschützten Bereich zu öffnen.

## Einrichten und aktualisieren

Server vor dem Update stoppen und wie üblich aktualisieren:

```sh
git switch beta
git pull --ff-only origin beta
go run . -dev -migrate
```

Anschließend „Administration“ öffnen. Es gibt genau einen festen Benutzer:

- Benutzername: **`admin`** (bereits vorausgefüllt).
- Startpasswort: **`Reifenapp-Start2026!`**.

Nach der ersten Anmeldung ist nur die Passwortänderung freigeschaltet. Eigenes
Passwort und Wiederholung eingeben; erst danach öffnet sich der Designer.
Mindestens 12 Zeichen, höchstens 72 Bytes (bei Umlauten können weniger als 72
Zeichen möglich sein). Startpasswort und bisheriges Passwort dürfen nicht erneut
gewählt werden. Für spätere Änderungen gibt es „Passwort ändern“; dort wird auch
das aktuelle Passwort verlangt.

Das eigene Passwort ersetzt den Startpasswort-Hash in der lokalen Datenbank.
Es bleibt nach Serverneustarts erhalten und wird ausschließlich als gesalzener
bcrypt-Hash gespeichert. Das Startpasswort wird nach der Änderung nicht mehr
akzeptiert. Alle alten Sitzungen werden beendet und die aktuelle Sitzung erneuert.
Das alte Verfahren über `REIFENAPP_ADMIN_PASSWORD` entfällt; diese Variable wird
nicht mehr verwendet. Es gibt keinen automatischen Rückfall auf das Startpasswort.

Für die normale Datenbank `-dev` weglassen. Entwicklung und normaler Betrieb haben
getrennte Datenbanken, damit auch getrennte Admin-Passwörter und Layouts. Die erste
Einrichtung einmal lokal vornehmen, bevor der Server gegebenenfalls im LAN
freigegeben wird. Passwörter oder Datenbankdateien nicht in GitHub veröffentlichen.
Bei einem Neustart werden bestehende Sitzungen ungültig; mit dem eigenen Passwort
neu anmelden. Ein Passwort-Reset ohne Kenntnis des aktuellen Passworts ist hier
nicht implementiert; keinesfalls dafür die Kundendatenbank löschen.

Migration `007_admin_account.sql` ergänzt das lokale Admin-Konto. Das explizite
Upgrade erstellt wie bisher vorher ein konsistentes SQLite-Backup nach
Integritätsprüfung. Bestehende PDF-Vorlagen, Kunden und Historieneinträge bleiben
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
- Raster: standardmäßig 5 mm, alternativ 1/2/10 mm oder „Aus“. Beim Ziehen
  rasten X/Y am Seitenraster ein; beim Skalieren die rechte/untere Kante.
  Neue und duplizierte Elemente starten ebenfalls auf Rasterpunkten.
- Pfeiltasten bewegen um einen Rasterschritt, mit Umschalt um zwei. Ohne Raster
  bleiben es 0,5 mm bzw. 2 mm. Ein Element neben dem Raster springt zuerst zum
  nächsten Rasterpunkt in Bewegungsrichtung.
- Alt beim Ziehen/Skalieren erlaubt vorübergehend freie Bewegung. Exakte manuelle
  Zahlenwerte in den Eigenschaften bleiben auch bei aktivem Raster möglich.
- Umschalten des Rasters verändert vorhandene Elemente nicht. Die A4-Seitengrenzen
  begrenzen Bewegungen auch dann, wenn die letzte mögliche Position kein
  Rasterpunkt ist. Das sichtbare Raster wird weder gespeichert noch gedruckt.
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
Die lokale Datenbank enthält ausschließlich den bcrypt-Hash des Passworts.
Das Konto wird einmalig mit erzwungenem Passwortwechsel angelegt; ein bestehendes
Konto wird beim Start nicht zurückgesetzt. Die Sitzungsrevision sperrt alte
Sitzungen auch dann, wenn eine andere laufende Instanz das Passwort geändert hat.
Für Netzwerkbetrieb den Transport-/Zugriffsschutz beim Bereitstellen
berücksichtigen; Standardstart bleibt auf 127.0.0.1 beschränkt.

Admin-Endpunkte:

- `POST /api/admin/login` mit `{ "username": "admin", "password": "…" }`.
  Session-Cookie und CSRF-Wert aus der Antwort benutzen. `must_change: true`
  sperrt alle Einstellungen bis zum Passwortwechsel.
- `GET /api/admin/session`; `POST /api/admin/logout`.
- `POST /api/admin/password` mit `new_password`, `confirm_password`, später
  zusätzlich `current_password`. Benötigt CSRF und erneuert Cookie/CSRF-Token.
- `GET /api/admin/layout?operation=storage` liefert gespeicherte Vorlage und
  Ausgangsvorlage. Andere Operationen: `wheel_change`, `reservation`, `release`.
- `POST /api/admin/layout` mit `operation`, `template`, `enabled`, `revision`.
- `POST /api/admin/preview` mit derselben Struktur liefert eine echte A4-PDF.
- `GET /api/admin/fields`, `/api/admin/values?operation=…` und `/api/admin/font`
  liefern Feldkatalog, fiktive Vorschauwerte und die Schrift.

Schreibzugriffe außer Login benötigen `X-CSRF-Token`. 401: nicht angemeldet,
403: Sitzung/Ursprung ungültig, 409: Revisionskonflikt, 422: Layout ungültig,
429: Anmeldebegrenzung. Layoutdaten und
PDFs werden nicht öffentlich zwischengespeichert.
