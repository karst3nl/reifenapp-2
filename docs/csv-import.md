# Alte CSV-Vorgänge importieren

Nach einem Update den Server beenden und mit `go run . -migrate` starten.
Migration 009 erweitert die Historie; die Anwendung sichert bestehende Datenbanken
vor einer Migration automatisch. Andere Prozesse mit Datenbankzugriff vorher beenden.

1. Administration öffnen und anmelden, dann **CSV-Import** wählen.
2. Die lokale Exportdatei auswählen und **Vorschau prüfen** anklicken.
3. Vorgänge, Dubletten und Prüfhinweise kontrollieren. Die Vorschau speichert nichts.
4. Die Archivübernahme bestätigen und den Import starten.

Vor jedem neuen Import erstellt die Anwendung eine geprüfte SQLite-Sicherung neben
der laufenden Datenbank (`data/*.backup-csv-*`). Der Import wird vollständig in einer
Transaktion gespeichert. Bei einem Fehler wird kein Teilimport übernommen.

## Datenerhaltung

Importierte Vorgänge sind in Schnellsuche und Historie mit **CSV-Import** gekennzeichnet.
Die Historie bietet eigene Filter für CSV-Quelle und Archivstatus. Auch Originalwerte
sind durchsuchbar. In der aufgeklappten Karte stehen die Originalspalten einschließlich
nicht erfasster Felder und Prüfhinweise. Die vollständige unveränderte Originaldatei
kann im Adminbereich erneut heruntergeladen werden und ist im Gesamtbackup enthalten.

Die Übernahme erstellt ausschließlich historische Vorgänge. Aus alten Ein- oder
Auslagerungen wird keine aktuelle Belegung abgeleitet; Kunden-, Fahrzeug-, Radsatz-
und Reservierungsbestände bleiben unverändert. Alte, heute ungültige Lagerplätze
bleiben als historische Angaben erhalten. CSV-Vorgänge zählen nicht zum aktuellen
Verschleißzähler oder Saisonfortschritt. Sie haben keine nachträglich erfundene PDF-Quittung.

Dieselbe Datei kann nicht doppelt importiert werden. Bei überlappenden Exporten werden
identische Vorgänge anhand aller Originalspaltenwerte nur einmal angelegt; jede neue
Datei mit ihren Originalzeilen bleibt dennoch erhalten. Unterschiedliche Schreibweisen
werden nicht automatisch zusammengeführt. Fehlende Schlussfelder und explizit leere
Schlussfelder gelten für diesen Vergleich als gleich.

## Unterstütztes Format

- Höchstens 10 MiB und 10.000 nichtleere Vorgänge pro Datei.
- Komma, Semikolon oder Tabulator; korrekt in Anführungszeichen gesetzte Werte werden unterstützt.
- UTF-8, Windows-1252 sowie UTF-16 mit BOM.
- Pflichtspalten: `DATUM`, `TÄTIGKEIT`, `NAME`, `KENNZEICHEN`, `LAGERPLATZ`.
- Optionale bekannte Spalten: `AUTO`, `REIFENART`, `FELGENART`, `PROFILTIEFEVORNE`,
  `PROFILTIEFEHINTEN`, `RADSCHRAUBEN`, `FELGENSCHLOSS`, `RADKAPPEN`, `HINWEISE`.
- Zusätzliche Spalten und verkürzte Datensätze bleiben erhalten. Nicht eindeutige
  Werte werden mit Prüfhinweisen und unverändertem Original gespeichert.
- Deutsche Datumsangaben mit optionaler Uhrzeit gelten als Zeit in Europe/Berlin;
  ISO-Daten werden ebenfalls erkannt. Unlesbare Daten bleiben als Original verfügbar,
  erhalten aber keinen erfundenen Zeitpunkt und erscheinen nicht in Datumsfiltern.
- Allwetter wird für die Anzeige als Ganzjahr eingeordnet; Profiltiefen mit `mm`
  und Dezimalkomma werden normalisiert, ohne auf halbe Millimeter zu runden.

Dateien mit fehlerhafter CSV-Struktur oder uneindeutiger Kopfzeile werden vollständig
abgewiesen. Vorschau zeigt höchstens 50 Zeilen; die Zähler berücksichtigen die ganze Datei.

CSV-Dateien enthalten personenbezogene Daten. Originalexporte, Backups und Laufzeitdaten
lokal geschützt aufbewahren und niemals ins Repository hochladen.
