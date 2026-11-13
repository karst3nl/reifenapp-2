# Vollständiges Datenbank-Backup

Unter **Administration → Backup** lässt sich eine vollständige Sicherung der
aktuell verwendeten Datenbank herunterladen. Anmeldung und abgeschlossenes
Admin-Passwort-Setup sind erforderlich.

## Herunterladen und ansehen

1. Unter „Aktuelle Datenbank“ prüfen, ob die normale oder die Entwicklungsdatenbank
   ausgewählt ist. Es wird immer nur die Datenbank des laufenden Servers gesichert.
2. **Vollständiges Backup herunterladen** anklicken und den Download abwarten.
3. Das ZIP-Paket im Dateimanager entpacken.
4. **uebersicht.html** im Browser öffnen. Dafür werden weder SQLite-Software noch
   eine laufende Reifenapp oder eine Internetverbindung benötigt.

Die Übersicht enthält alle Tabellen mit deutschen Überschriften, Kundendaten,
Fahrzeugdaten, Reifenangaben, Zubehör, Belegungen, Reservierungen und Historie.
Strukturierte Angaben lassen sich aufklappen. Inhalte werden als Text ausgegeben;
Kundennotizen werden nicht als HTML oder JavaScript ausgeführt.

## Inhalt des ZIP-Pakets

| Datei | Zweck |
| --- | --- |
| `datenbank.sqlite` | Vollständige, eigenständig nutzbare SQLite-Kopie mit allen Daten, Historie, Einstellungen, PDF-Vorlagen/Logos und Admin-Passwort-Hash |
| `uebersicht.html` | Offline lesbare Übersicht für den Browser |
| `daten.json` | Textlesbarer Export aller Tabellen; der Passwort-Hash ist ausgeblendet, binäre Werte werden als Base64 dargestellt |
| `schema.sql` | Tabellen, Indizes, Ansichten und Trigger des gesicherten Schemas |
| `manifest.json` | Zeitpunkt in UTC, Quelldateiname, Schemaversion, Zeilenzahlen und SHA-256-Prüfsumme der SQLite-Datei |
| `receipt-header.json` | Zum Sicherungszeitpunkt im Server geladener Firmenkopf |
| `BITTE-LESEN.txt` | Kurzanleitung zum Öffnen und Wiederherstellen |

Die SQLite-Datei ist die Grundlage für eine vollständige Wiederherstellung. HTML
und JSON sind zusätzliche Leseexporte; sie ersetzen die Datenbankkopie nicht.
Die Sicherung ist unverschlüsselt und enthält Kundendaten. Sie gehört auf einen
geschützten Datenträger und nicht in GitHub. Dateiname und entpackter Paketordner
sind zusätzlich in `.gitignore` ausgeschlossen.

## Konsistenz und laufender Betrieb

Die Sicherung wird mit `VACUUM INTO` erstellt. Dadurch enthält sie auch bereits
bestätigte Änderungen aus dem WAL und ist ohne zusätzliche `-wal`-/`-shm`-Dateien
nutzbar. Der Server kann weiterlaufen. Während SQLite den Stand kopiert, können
andere Vorgänge im selben Prozess kurz warten.

Integritätsprüfung, Exporte und Prüfsumme beziehen sich auf **dieselbe Kopie**,
nicht auf nacheinander abgefragte Live-Tabellen. Neue Buchungen nach dem
Sicherungszeitpunkt sind erst in einer späteren Sicherung enthalten.

Gesichert werden gespeicherte Daten. Ungespeicherte PDF-Entwürfe werden weder
übernommen noch verworfen. Entwicklung und normaler Betrieb bleiben getrennt.
Die Sicherung verändert die laufende Datenbank nicht. Temporäre Dateien werden
nach dem Download, einem Fehler oder einem abgebrochenen Auftrag entfernt.
Pro Serverprozess läuft höchstens ein Backup-Download gleichzeitig.

## Wiederherstellen in einem neuen Projektordner

Die Downloadfunktion überschreibt keine vorhandene Datenbank und bietet keinen
automatischen Wiederherstellungsbutton. Zum Prüfen oder Wiederherstellen einen
**neuen, leeren Projektordner** verwenden. Dort darf noch kein Server laufen und
keine Datenbank vorhanden sein.

Beispiel für ein Backup der normalen Datenbank:

```sh
git clone --branch beta https://github.com/karst3nl/reifenapp-2.git Reifenapp-Wiederherstellung
cd Reifenapp-Wiederherstellung
mkdir -p data local
cp -n /pfad/zum/entpackten-backup/datenbank.sqlite data/reifenapp.sqlite
cp -n /pfad/zum/entpackten-backup/receipt-header.json local/receipt-header.json
go run .
```

Die Beispielpfade vorher durch die eigenen Pfade ersetzen. `cp -n` überschreibt
keine vorhandenen Dateien; bei bereits vorhandener Datenbank einen anderen leeren
Ordner verwenden. Die bisherige Installation und ihre Daten bleiben unangetastet.
Keine einzelne Datei über eine bestehende Datenbank mit alten WAL-/SHM-Dateien kopieren.

Für ein **Entwicklungsbackup** stattdessen nach `data/development.sqlite` kopieren
und `go run . -dev` starten. Fiktive Entwicklungsdaten nicht in die produktive
Installation übernehmen. Der Quelldateiname steht im Manifest und im Adminbereich.

Die App-Version muss das gesicherte Schema unterstützen. Falls ein Upgrade nötig
ist, den Server stoppen und das dokumentierte `-migrate`-Verfahren verwenden; dabei
wird vor Änderungen wiederum ein konsistentes Backup angelegt. Das Admin-Passwort
entspricht dem Stand der Sicherung. Ein Rücksetzen auf das Startpasswort findet
nicht statt. Nach dem Start Suche, Historie, Lagerplätze und PDF-Vorlagen prüfen.

## API und Prüfungen

- `GET /api/admin/backup/info`: Quelldateiname ohne vollständigen Pfad und Schemaversion.
- `POST /api/admin/backup` mit JSON `{}`: ZIP-Download, Admin-Sitzung und `X-CSRF-Token` nötig.
- Antworten sind nicht cachebar. Gleichzeitiger Backup-Auftrag: HTTP 429;
  Fehler beim Erstellen: HTTP 500 ohne internen Datenbankpfad.
- Der Server gibt erst nach erfolgreicher Erstellung und Integritätsprüfung ein
  ZIP als Download aus. Bei Fehlern kann der Auftrag erneut gestartet werden.

Automatisierte Tests prüfen insbesondere WAL-Inhalte, Historie, Vorlagen und Logos,
Admin-Konto, Prüfsumme, Exportkonsistenz bei parallelen Schreibzugriffen, Öffnen der
wiederhergestellten Datenbank, Berechtigungen, CSRF-Schutz und temporäre Bereinigung.
Für die Funktion ist keine neue Migration und keine zusätzliche Go-Abhängigkeit nötig.
