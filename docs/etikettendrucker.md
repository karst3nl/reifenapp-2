# USB-Etikettendrucker: ARGOX OS-214EX PRO

Die App erstellt eine einseitige PDF von **78 × 51 mm** und übergibt sie auf dem
App-Server ohne Browser-Druckdialog an eine lokale CUPS-USB-Warteschlange.
Der vorhandene A4-Beleg wird weiterhin unabhängig erstellt und im Browser geöffnet.
Es werden keine neuen Go-Abhängigkeiten benötigt.

## Drucker auf CachyOS einrichten

1. Drucker per USB an den Rechner anschließen, auf dem `go run .` läuft.
   Etiketten und passendes Transferfarbband nach Gerätehandbuch einlegen und
   den Etikettensensor kalibrieren.
2. CUPS und PDF-Filter installieren und den Dienst starten:

   ```bash
   sudo pacman -S cups cups-filters
   sudo systemctl enable --now cups.service
   ```

3. Den passenden Linux-Treiber von der
   [Argox-Produktseite](https://www.argox.com/products-detail/os-214ex_pro-os-200_pro/)
   nach Herstelleranleitung installieren. In den Linux-Druckereinstellungen oder
   [lokalem CUPS](http://localhost:631) eine **USB-Warteschlange mit Treiber** anlegen,
   beispielsweise `Argox_USB`. Keine Raw-Warteschlange verwenden: Die App sendet PDF,
   keine PPLA/PPLB/PPLZ-Rohbefehle. Die Treiber müssen PDF in die zum Gerät passende
   Druckersprache umwandeln können.
4. Im Treiber 78 mm Breite, 51 mm Vorschublänge und das verwendete Material einstellen:
   Thermotransfer, Etikettenabstand/Sensor, Druckgeschwindigkeit und Schwärzung passend
   zur Rolle und zum Farbband. Diese Geräteeinstellungen erfolgen in CUPS bzw. am Gerät.
5. Im Terminal unter demselben Linux-Benutzer wie die App prüfen:

   ```bash
   lpstat -h localhost -v
   lpstat -h localhost -p Argox_USB -l
   lpoptions -h localhost -p Argox_USB -l
   ```

Die Geräte-URI muss mit `usb://` beginnen. Die App listet bewusst nur lokale
USB-Warteschlangen auf. Dass eine Warteschlange existiert, bestätigt noch nicht,
dass der Drucker eingeschaltet, mit Papier versorgt oder fehlerfrei verbunden ist.
Der Server braucht die Befehle `lpstat` und `lp` im PATH und Druckberechtigung.
Die App läuft weiterhin ohne Root-Rechte. Sie installiert oder löscht keine
Systemdrucker und ändert keine CUPS-Systemeinstellungen.

## In der App konfigurieren

Nach dem Update Server beenden und `go run . -migrate` starten. Migration 011 wird
mit automatischer Datenbanksicherung eingespielt. Anschließend:

1. **Administration → Etikettendrucker** öffnen und Druckerliste aktualisieren.
2. USB-Warteschlange und 1–8 Exemplare je Vorgang auswählen. Standard ist ein Etikett.
   Für vier einzelne Räder können vier Exemplare eingestellt werden.
3. Text- und Datenfelder konfigurieren: X/Y ab linker oberer Ecke, Breite/Höhe in mm,
   Schriftgröße in pt und Fettdruck. Maximal 20 Elemente innerhalb von 78 × 51 mm.
   Ein Präfix kann vor ein Datenfeld gesetzt werden; „Freitext“ enthält nur eigenen Text.
   Elemente dürfen sich bewusst überlagern; Positionen in der Vorschau kontrollieren.
4. Die PDF-Vorschau mit fiktiven Daten ansehen und die Einstellungen speichern.
5. **Ein Testetikett drucken** und Größe, Orientierung, Ränder und Lesbarkeit am
   tatsächlichen Gerät prüfen. Ein Testdruck erzeugt immer genau ein fiktives Etikett.
6. Automatik für **Neueinlagerung** und/oder **Radwechsel** aktivieren und speichern.
   Sie ist im Auslieferungszustand deaktiviert. Reservierungen, Auslagerungen,
   CSV-Importe und das Öffnen von Belegen lösen keinen Etikettendruck aus.

Die App verwendet `media=Custom.78x51mm`, `print-scaling=none`, einseitigen Druck,
keine Bannerseiten und eine unveränderte 78 × 51-mm-PDF. Falls der Treiber das
benutzerdefinierte Format nicht korrekt unterstützt, muss die Warteschlange vor
Aktivierung der Automatik entsprechend eingerichtet werden. Der Endabgleich
am physischen Drucker ist erforderlich; die Cloud-Entwicklungsumgebung hat keinen
Zugriff auf den USB-Drucker in der Werkstatt.

## Aufträge und Fehler

Ein automatischer Auftrag wird **in derselben SQLite-Transaktion wie der Vorgang**
gespeichert. Bei abgewiesenen Formularen entsteht kein Druckauftrag. Jeder Vorgang
hat höchstens einen automatischen Etikettenauftrag. Neue Vorlagenänderungen wirken
nicht rückwirkend auf bereits vorgemerkte Aufträge.

Ein Hintergrundprozess übergibt etwa alle zwei Sekunden einen Auftrag an CUPS.
Nach dem Absenden zeigt die Startseite den Druckstatus separat an; nach längerer
Wartezeit verweist sie auf die Adminübersicht. Der gespeicherte Werkstattvorgang
bleibt bei einem Druckerfehler erhalten. Der Adminbereich zeigt die letzten 100
Aufträge und aktualisiert die Liste alle fünf Sekunden.

- **Vorgemerkt:** Noch nicht übergeben; kann in der App abgebrochen werden.
- **Übergabe läuft:** Der Hintergrundprozess arbeitet daran.
- **An CUPS übergeben:** CUPS hat eine Auftragsnummer bestätigt. Das ist ausdrücklich
  keine Bestätigung des physischen Ausdrucks. Wartende, angehaltene oder fehlerhafte
  CUPS-Aufträge unter localhost:631 kontrollieren und dort verwalten.
- **Fehlgeschlagen:** Vor der Übergabe gescheitert, etwa weil Drucker oder Vorlage
  nicht verfügbar sind. Vollständiger Text wird passend verkleinert; passt er auch
  bei 6 pt nicht ins Feld, scheitert der Auftrag, statt Angaben still abzuschneiden.
- **Übergabe unklar:** Timeout, nicht bestätigte Auftragsnummer oder Serverabbruch
  während der Übergabe. Vor einem weiteren Versuch CUPS und vorhandene Etiketten prüfen.
- **Abgebrochen:** Noch ausstehender Auftrag wurde manuell verworfen.

Es gibt bewusst **keine automatischen Wiederholungen** fehlgeschlagener oder unklarer
Übergaben: Ein Drucker kann ein Etikett ausgegeben haben, bevor die App die Bestätigung
verlor. „Erneut versuchen“ benötigt eine ausdrückliche Bestätigung und verwendet die
aktuell gespeicherte Warteschlange/Vorlage mit den ursprünglichen Vorgangsdaten.
Bereits an CUPS übergebene Aufträge können in der App nicht erneut abgeschickt werden.

Beim Neustart werden ausstehende Aufträge weiterverarbeitet; unterbrochene Übergaben
werden als unklar markiert. Die Automatik-Einstellung steuert neue Vorgänge; bereits
vorgemerkte Aufträge bleiben erhalten und können einzeln abgebrochen werden.
Nur **einen App-Server pro Datenbank** betreiben. Beim Wiederherstellen eines älteren
Backups vor Anschluss des Druckers die darin enthaltenen ausstehenden Aufträge prüfen,
da sie im ursprünglichen System inzwischen bereits gedruckt worden sein können.
Datenbankbackup enthält Konfiguration und App-Aufträge, nicht die CUPS-Konfiguration
oder den Zustand des Geräts. Auch CUPS speichert Druckdaten lokal; entsprechend schützen.

## Raspberry Pi und lokale Netznutzung

Beim späteren Pi-Betrieb muss der Drucker am Pi angeschlossen und dort in CUPS
eingerichtet sein. Den zur Architektur des Betriebssystems passenden Treiber verwenden;
Argox nennt auch Raspberry-Pi-Treiber auf seiner Produktseite. Unterstützung der
konkreten Betriebssystem-/Treiberkombination am Gerät prüfen. Browser im lokalen Netz
lösen den Druck auf dem App-Server aus, nicht an ihrem eigenen USB-Anschluss.
Keine Cloudverbindung für den laufenden Druck erforderlich.

Technische Referenzen: [CUPS-Druckoptionen](https://www.cups.org/doc/options.html),
[Argox-Handbuch und Treiber](https://www.argox.com/products-detail/os-214ex_pro-os-200_pro/).
