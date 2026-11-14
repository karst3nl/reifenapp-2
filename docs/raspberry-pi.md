# Raspberry Pi: Werkstattbetrieb nur über SSH

Diese Anleitung installiert die App als systemd-Dienst: Sie startet nach dem
Einschalten automatisch, läuft unabhängig von einer SSH-Sitzung und ist im lokalen
Werkstattnetz erreichbar. Updates erfolgen bewusst per SSH aus dem GitHub-Branch
**main**, mit Tests, Datenbanksicherung und anschließender Versionsprüfung.

Die Dateien sind für Raspberry Pi OS Lite mit systemd vorbereitet. Ein tatsächlicher
Pi-Neustart, die ARM64-Hardware und der USB-Drucker müssen vor Ort geprüft werden;
die Entwicklungsprüfungen ersetzen diesen Test nicht.

## 1. Pi und SSH vorbereiten

Empfohlen: Raspberry Pi 4/5 mit mindestens 2 GB RAM, zuverlässiger Stromversorgung
und SSD oder hochwertiger Speicherkarte; **Raspberry Pi OS Lite (64-bit)**.
Auf einer neuen Speicherkarte Raspberry Pi Imager verwenden und dort vor dem ersten
Start konfigurieren:

- Eigenen SSH-Benutzer mit Passwort und/oder SSH-Schlüssel einrichten.
- SSH aktivieren, Hostname z. B. `reifenapp`.
- Bei WLAN die Werkstatt-Zugangsdaten konfigurieren; Ethernet ist für den Dauerbetrieb
  vorzuziehen. Ein vorhandenes eingerichtetes Betriebssystem nicht neu aufspielen.

Vom CachyOS-PC aus verbinden, dabei `werkstatt` durch deinen eingerichteten
SSH-Benutzernamen ersetzen:

```bash
ssh werkstatt@reifenapp.local
```

Falls `.local` nicht funktioniert, die IP-Adresse in der Geräteliste des Routers
nachsehen und `ssh werkstatt@192.168.1.50` verwenden. Alle weiteren Installations-
und Updatebefehle laufen in dieser SSH-Sitzung auf dem Pi. Die App selbst benötigt
keinen Desktop, Bildschirm oder dauerhaft geöffnetes Terminal.

```bash
uname -m
sudo apt update
sudo apt install -y git golang-go build-essential sqlite3 curl ca-certificates python3 avahi-daemon tmux
sudo systemctl enable --now ssh avahi-daemon
go version
```

Für 64-bit-Pi-OS muss `uname -m` **aarch64** melden. Go muss mindestens die Version
aus `go.mod` erfüllen (derzeit 1.23). Eine aktuelle gepflegte Go-Version verwenden.
Ist das Paket auf einem älteren Pi-OS zu alt, vor dem Build Go nach der
[offiziellen Linux-ARM64-Anleitung](https://go.dev/doc/install) aktualisieren oder
auf einem neuen Datenträger ein aktuelles Pi-OS einrichten. Kein x86/amd64-Go-Paket
auf dem Pi verwenden. Die SQLite-Anbindung verwendet CGO, deshalb wird **auf dem Pi
selbst** mit C-Compiler gebaut; ein einfaches `GOARCH=arm64` auf dem PC genügt nicht.

## 2. Feste Adresse im Werkstattnetz

Im Router für den verwendeten Netzwerkadapter eine **DHCP-Reservierung** einrichten.
So erhält der Pi nach jedem Neustart wieder dieselbe IP. Beispiele hier verwenden
`192.168.1.50`; deine tatsächliche Adresse kann anders sein.

```bash
hostname -I
ip -br link
```

Die MAC-Adresse des tatsächlich verwendeten Ethernet- oder WLAN-Adapters im Router
zuordnen. Der Hostname `reifenapp.local` ist zusätzlich bequem, funktioniert aber
nicht in jedem Netzwerksegment. Die reservierte IP ist der verlässliche Ersatz.
Beim Wechsel zwischen Ethernet und WLAN gelten unterschiedliche MAC-Adressen.
Nicht während der einzigen SSH-Verbindung unvorbereitet die Netzwerkkonfiguration
umschreiben. Eine Router-Reservierung benötigt keine solche Änderung am Pi.

Die App hört auf `0.0.0.0:8080`, also auf allen IPv4-Schnittstellen. Sie bleibt auch
bei späterem Netzwerkaufbau oder erneutem Verbinden erreichbar, sobald der Pi wieder
im Netz ist. Stromausfall, ausgefallenes Netzwerk oder ein voller Datenträger können
natürlich keine Software-Einstellung ausgleichen.

## 3. Code, Binärdatei und Daten getrennt installieren

| Ort | Inhalt / Eigentümer |
| --- | --- |
| `/opt/reifenapp/source` | Git-Checkout, gehört deinem SSH-Benutzer |
| `/opt/reifenapp/bin/reifenapp` | Installierte Binärdatei, gehört root |
| `/var/lib/reifenapp/data/reifenapp.sqlite` | Produktionsdatenbank, gehört dem Dienstbenutzer `reifenapp` |
| `/var/lib/reifenapp/local/` | Optionaler Firmenkopf und weitere lokale Dateien |
| `/etc/reifenapp/reifenapp.env` | Lokale Listen-Adresse / Port |
| `/var/backups/reifenapp/` | Geschützte Sicherungen vor Updates |

Kundendaten liegen damit **außerhalb des Git-Checkouts**. Der Dienst führt keinen
Go-Build aus und läuft ohne Root-Rechte. Den Dienstbenutzer nicht für SSH verwenden.

Die folgende Einrichtung ist für eine **neue Installation**. Wenn diese Pfade
bereits eine produktive Installation enthalten, nicht erneut Dateien darüber
installieren, sondern mit Abschnitt „Updates“ fortfahren bzw. erst den vorhandenen
Aufbau prüfen. Insbesondere keine bestehende Datenbank löschen oder mit Beispielen füllen.

```bash
sudo useradd --system --user-group --home-dir /var/lib/reifenapp --shell /usr/sbin/nologin reifenapp
sudo install -d -m 0755 /opt/reifenapp
sudo install -d -m 0755 -o "$(id -un)" -g "$(id -gn)" /opt/reifenapp/source
git clone --branch main https://github.com/karst3nl/reifenapp-2.git /opt/reifenapp/source
cd /opt/reifenapp/source
export CGO_ENABLED=1
go mod download
go test ./...
go vet ./...
REIFENAPP_COMMIT=$(git rev-parse HEAD)
REIFENAPP_BUILD_DIR=$(mktemp -d)
go build -mod=readonly -trimpath -ldflags "-X main.version=$REIFENAPP_COMMIT" -o "$REIFENAPP_BUILD_DIR/reifenapp" .
sudo install -d -m 0755 /opt/reifenapp/bin
sudo install -m 0755 "$REIFENAPP_BUILD_DIR/reifenapp" /opt/reifenapp/bin/reifenapp
sudo install -d -m 0700 -o reifenapp -g reifenapp /var/lib/reifenapp
sudo install -d -m 0700 -o reifenapp -g reifenapp /var/lib/reifenapp/data /var/lib/reifenapp/local
```

Vor dem ersten Serverstart die **leere Produktionsdatenbank** initialisieren:

```bash
sudo -u reifenapp -- bash -c 'cd /var/lib/reifenapp && /opt/reifenapp/bin/reifenapp -migrate-only'
```

`-migrate-only` führt nur Datenbankinitialisierung/Migration aus und beendet sich.
Es öffnet keinen HTTP-Port und startet keinen Etikettendruck. Bei einer bestehenden
Datenbank erfolgen Integritätsprüfung und nötigenfalls ein automatisches Backup vor
Schemaänderungen. **Kein `-dev` im produktiven Dienst verwenden.**

### Vorhandene Werkstattdaten vom PC übernehmen

Diesen optionalen Schritt vor dem ersten Pi-Serverstart durchführen. Auf dem
bisherigen PC über **Administration → Backup** die vollständige Sicherung herunterladen,
entpacken und den bisherigen Server danach beenden, damit nicht zwei getrennte
Datenbestände weitergeführt werden.

Die SQLite-Datei aus dem Backup enthält auch Historie, Vorlagen, Logos, Admin-Passwort,
CSV-Archiv und Etikettenkonfiguration. CUPS und die Linux-Druckerwarteschlange müssen
auf dem Pi separat eingerichtet werden.

Vom PC die entpackte `datenbank.sqlite` hochladen:

```bash
scp /pfad/zum/entpackten-backup/datenbank.sqlite werkstatt@reifenapp.local:~/reifenapp-uebernahme.sqlite
```

Auf dem Pi darf der Dienst noch nicht laufen. Wenn die leere Datenbank oben bereits
initialisiert wurde, diese **als kompletten Datenordner aufbewahren**, nicht einzelne
WAL-/SHM-Dateien darübermischen. Der Zielname muss neu sein:

```bash
sudo test ! -e /var/lib/reifenapp/data-vor-uebernahme &&
sudo mv /var/lib/reifenapp/data /var/lib/reifenapp/data-vor-uebernahme &&
sudo install -d -m 0700 -o reifenapp -g reifenapp /var/lib/reifenapp/data &&
sudo install -m 0600 -o reifenapp -g reifenapp "$HOME/reifenapp-uebernahme.sqlite" /var/lib/reifenapp/data/reifenapp.sqlite &&
sudo -u reifenapp -- bash -c 'cd /var/lib/reifenapp && /opt/reifenapp/bin/reifenapp -migrate-only'
```

Jede Zeile nur fortsetzen, wenn die vorherige erfolgreich war. Einen optionalen
`receipt-header.json` ebenfalls aus der Sicherung hochladen und als
`/var/lib/reifenapp/local/receipt-header.json` mit Eigentümer `reifenapp` und Modus
0600 installieren. Quelle und bisherige Daten unverändert aufbewahren.

Vor dem Anschließen des Druckers die übernommenen ausstehenden Druckaufträge prüfen:
Sie könnten auf dem alten PC inzwischen gedruckt worden sein. Die App setzt das
Admin-Passwort nicht zurück. Die ausführliche [Backup-Anleitung](backups.md) erklärt
das Sicherungsformat und die Wiederherstellung.

## 4. Automatischen Start per systemd aktivieren

```bash
cd /opt/reifenapp/source
sudo install -d -m 0755 /etc/reifenapp
sudo install -m 0644 deploy/reifenapp.env.example /etc/reifenapp/reifenapp.env
sudo install -m 0644 deploy/reifenapp.service /etc/systemd/system/reifenapp.service
sudo systemctl daemon-reload
sudo systemctl enable --now reifenapp.service
sudo systemctl status reifenapp.service --no-pager
curl --fail http://127.0.0.1:8080/api/health
```

Erwartet: Dienst `active (running)` und JSON mit `status: "ok"` sowie dem Git-Commit
als `version`. Die App startet beim Pi-Neustart automatisch. Bei einem unerwarteten
Prozessabbruch versucht systemd alle fünf Sekunden einen Neustart. Nach einem
bewussten `systemctl stop` bleibt sie bis zum Startbefehl oder nächsten Boot gestoppt.

Vom Werkstatt-PC öffnen:

- `http://reifenapp.local:8080`
- oder `http://192.168.1.50:8080` mit deiner reservierten IP.

Bei einer neuen Datenbank sofort den Adminbereich öffnen: Benutzer `admin`,
Startpasswort `Reifenapp-Start2026!`; beim ersten Login ein eigenes Passwort festlegen.
Bei übernommenen Daten gilt das bisherige Passwort.

**Neustart tatsächlich prüfen**, wenn gerade keine Buchungen oder Druckübergaben laufen:

```bash
sudo reboot
```

Die SSH-Verbindung endet. Nach dem Boot erneut verbinden und prüfen:

```bash
ssh werkstatt@reifenapp.local
systemctl is-enabled reifenapp.service
systemctl is-active reifenapp.service
curl --fail http://127.0.0.1:8080/api/health
```

Zusätzlich vom Werkstatt-PC eine Suche und die Historie öffnen. Der Dienst läuft
dabei unabhängig von SSH. Kein `nohup`, `screen`, Desktop-Autostart oder `go run` nötig.

## 5. Updates aus GitHub per SSH

Updates sind **manuell ausgelöst**, nicht ungeprüft bei jedem Boot. Außerhalb laufender
Buchungen aktualisieren; ein neuer Build kann auf dem Pi einige Minuten benötigen.
Für eine unterbrechungssichere SSH-Sitzung `tmux` verwenden:

```bash
tmux new -s reifenapp-update
bash /opt/reifenapp/source/deploy/update.sh
```

Falls die Verbindung abbricht, später `tmux attach -t reifenapp-update` verwenden.
Mit Strg+B und anschließend D lässt sich die Sitzung bewusst verlassen.
Das Skript als normalen SSH-Benutzer mit sudo-Recht starten, **nicht** mit `sudo bash`.
Eine öffentliche GitHub-Adresse benötigt keinen Token und kein GitHub-Passwort.

Das Update:

1. Prüft Repo-Adresse, Branch `main`, unveränderten Checkout und vorhandene Produktivdaten.
2. Holt `origin/main` und aktualisiert ausschließlich per Fast-forward. Lokale Änderungen
   oder abweichende Historien werden nicht zurückgesetzt oder überschrieben.
3. Führt Go-Tests und `go vet` aus und baut nativ eine Binärdatei mit Commit-Kennung.
   Bis dahin bleibt der bisherige Dienst aktiv. Go-Abhängigkeiten benötigen ggf. Internet.
4. Stoppt den Dienst kontrolliert, wartet auf Anfragen/Druckstatus und sichert die SQLite-
   Datenbank mit der SQLite-Backup-API einschließlich bestätigter WAL-Daten.
5. Prüft die Sicherung und bewahrt bisherige Binärdatei und `local/` mit auf.
6. Führt Migrationen mit der neuen Version **ohne** HTTP- oder Druckstart aus und tauscht
   die Binärdatei danach atomar aus.
7. Startet den Dienst und prüft per HTTP die Datenbankverfügbarkeit und **genau den neuen
   Commit**. Der Pfad der Sicherung wird ausgegeben.

Schema-Upgrades erstellen nötigenfalls zusätzlich ein eigenes Backup neben der Datenbank.
Das Update stellt niemals automatisch eine alte Datenbank wieder her: Nach einem
fehlgeschlagenen Start könnten bereits neue Daten geschrieben worden sein. Vor Beginn
der Migration kann das Skript den bisherigen Dienst bei einem Fehler wieder starten;
ab Beginn der Migration bleibt der Dienst bei einem Fehler gestoppt, bis die Ursache
geklärt ist. Code-Checkout und laufende Binärdatei können nach einem fehlgeschlagenen
Build/Update unterschiedliche Versionen haben; `/api/health` und `-version` zeigen die
laufende bzw. installierte Version.

Unit, `/etc/reifenapp` und CUPS werden bei Updates **nicht** überschrieben. Falls sich
diese Dateien künftig ändern müssen, die Änderungen im Repo prüfen und gezielt übernehmen.
Sicherungen werden nicht automatisch gelöscht. Freien Speicher mit `df -h` kontrollieren
und Sicherungen zusätzlich auf einen anderen Datenträger übertragen. Ein Backup nur
auf derselben SD-Karte schützt nicht vor deren Ausfall.

### Bestehende Beta-Installation auf main umstellen

Eine bereits nach dieser Anleitung eingerichtete Beta-Installation behält ihre
Daten unter `/var/lib/reifenapp`. Zuerst prüfen, dass der Code-Checkout keine
ungesicherten Änderungen enthält. Danach als SSH-Benutzer:

```bash
cd /opt/reifenapp/source
git status --short
git fetch origin
git switch main
git merge --ff-only origin/main
bash deploy/update.sh
```

Bei lokalen Änderungen oder Git-Konflikten stoppen und diese klären; kein
`reset --hard` verwenden. Der bisherige Dienst läuft bis zur vorgesehenen
Update-Unterbrechung weiter. Für bewusste Entwicklung auf `beta` ist weiterhin
`REIFENAPP_BRANCH=beta bash deploy/update.sh` möglich, wenn auch `beta`
ausgecheckt ist. Für den Werkstattbetrieb bleibt `main` der Standard.

## 6. Betrieb, Logs und Portänderung

```bash
sudo systemctl status reifenapp.service --no-pager
sudo journalctl -u reifenapp.service -n 100 --no-pager
sudo journalctl -u reifenapp.service -f
sudo systemctl restart reifenapp.service
sudo systemctl stop reifenapp.service
sudo systemctl start reifenapp.service
/opt/reifenapp/bin/reifenapp -version
df -h
```

Zum Ändern der Adresse oder des Ports:

```bash
sudo nano /etc/reifenapp/reifenapp.env
sudo systemctl restart reifenapp.service
```

Beispiel: `REIFENAPP_ADDR=0.0.0.0:8090`. Für das Update dann denselben Port angeben:

```bash
REIFENAPP_HEALTH_URL=http://127.0.0.1:8090/api/health bash /opt/reifenapp/source/deploy/update.sh
```

Wenn eine Firewall aktiv ist, Port 8080 bzw. deinen gewählten Port nur für das
Werkstattnetz freigeben. SSH-Zugang nicht versehentlich sperren. Keine Router-
Portweiterleitung ins Internet einrichten. Der Adminbereich hat einen Login,
die normalen Werkstattfunktionen derzeit nicht; der Zugriff gehört ins vertraute
Werkstattnetz. CUPS muss nicht im LAN freigegeben werden.

Wenn der lokale `curl`-Aufruf funktioniert, die App am Werkstatt-PC aber nicht
erreichbar ist, zuerst die reservierte IP, dasselbe LAN, Firewall und eine mögliche
Gastnetz-/Client-Isolation am WLAN-Router prüfen. Ein `.local`-Namensproblem mit der
direkten IP ausschließen.

## 7. Etikettendruck auf dem Pi über SSH einrichten

```bash
sudo apt install -y cups cups-filters
sudo systemctl enable --now cups.service
lpstat -h localhost -v
```

Den zum Pi-OS passenden **ARM64-Treiber** des Argox verwenden, nicht den x86-Treiber.
Der Drucker muss per USB am **Pi** hängen. Danach die Schritte unter
[Etikettendrucker](etikettendrucker.md) für Warteschlange, 78 × 51 mm, Kalibrierung
und Testdruck ausführen. In jener Anleitung gelten die `pacman`-Befehle nur für
CachyOS; auf dem Pi werden die oben genannten `apt`-Pakete verwendet.

Eine grafische CUPS-Weboberfläche lässt sich bei Bedarf ohne Pi-Desktop per SSH-Tunnel
am Werkstatt-PC öffnen. Auf dem PC eine weitere SSH-Verbindung starten:

```bash
ssh -N -L 8631:localhost:631 werkstatt@reifenapp.local
```

Dann am PC `http://localhost:8631` öffnen. Das ist die lokale CUPS-Oberfläche des Pi.
Druckerinstallation bzw. Geräteverwaltung benötigt die passenden CUPS-Administratorrechte;
dafür den SSH-Benutzer nach CUPS-/Treiberanleitung berechtigen, nicht den App-Dienst.

Die Druckberechtigung des App-Benutzers prüfen:

```bash
sudo -u reifenapp -- lpstat -h localhost -p Argox_USB -l
```

Zum Testen den Testdruck im Adminbereich verwenden. Der systemd-Dienst darf `lp` und
`lpstat` ausführen und erreicht lokales CUPS. Die Automatik erst nach einem erfolgreichen
physischen Test aktivieren. Fehlende Drucker dürfen den App-Start nicht verhindern.

## 8. Fehler und kontrollierter Rückfall

Bei einem fehlgeschlagenen Update zunächst `journalctl` lesen und die Ursache prüfen
(z. B. Speicherplatz, falscher Port, fehlende Bibliothek oder Schemaversion). Häufig lässt
sich die neue Version nach Behebung mit `systemctl start reifenapp` wieder starten,
ohne Daten wiederherzustellen. Eine alte Binärdatei darf nicht ungeprüft mit einer
bereits migrierten Datenbank gestartet werden.

Für einen **bewussten Rückfall auf den Stand vor einem Update** müssen passende alte
Binärdatei und Vor-Update-Datenbank gemeinsam verwendet werden. Dadurch fehlen spätere
Buchungen in der wiederhergestellten Datenbank; den aktuellen Zustand deshalb vorher
vollständig aufbewahren. Wenn nach dem Update schon gearbeitet wurde, diese Daten erst
prüfen/exportieren und den Rückfall nicht blind durchführen.

Beispiel nach Prüfung der Sicherung, während kein zweiter App-Prozess läuft:

```bash
sudo systemctl stop reifenapp.service
sudo ls -la /var/backups/reifenapp
```

Den richtigen Sicherungsordner und **neue, noch nicht vorhandene** Aufbewahrungsnamen wählen:

```bash
# Beispielnamen durch den tatsächlichen Zeitstempel des Updates ersetzen.
REIFENAPP_BACKUP=/var/backups/reifenapp/20261113T120000-1234
REIFENAPP_KEPT_STATE=/var/lib/reifenapp-zustand-vor-rueckfall
sudo test -d "$REIFENAPP_BACKUP"
sudo test ! -e "$REIFENAPP_KEPT_STATE"
sudo sqlite3 -readonly "$REIFENAPP_BACKUP/reifenapp.sqlite" 'PRAGMA integrity_check;'
```

Nur fortsetzen, wenn die Integritätsprüfung `ok` meldet und die Pfade stimmen.
Alle folgenden Schritte einzeln prüfen und bei Fehler stoppen:

```bash
sudo mv /var/lib/reifenapp "$REIFENAPP_KEPT_STATE"
sudo install -d -m 0700 -o reifenapp -g reifenapp /var/lib/reifenapp /var/lib/reifenapp/data
sudo install -m 0600 -o reifenapp -g reifenapp "$REIFENAPP_BACKUP/reifenapp.sqlite" /var/lib/reifenapp/data/reifenapp.sqlite
# Nur falls der Sicherungsordner ein local/-Verzeichnis enthält:
sudo cp -a "$REIFENAPP_BACKUP/local" /var/lib/reifenapp/local
sudo chown -R reifenapp:reifenapp /var/lib/reifenapp
sudo install -m 0755 "$REIFENAPP_BACKUP/reifenapp.previous" /opt/reifenapp/bin/reifenapp.restore
sudo mv -T /opt/reifenapp/bin/reifenapp.restore /opt/reifenapp/bin/reifenapp
sudo systemctl start reifenapp.service
sudo systemctl status reifenapp.service --no-pager
curl --fail http://127.0.0.1:8080/api/health
```

Den vorherigen Zustand samt WAL-/SHM-Dateien unter `REIFENAPP_KEPT_STATE` aufbewahren.
Kein Mischen einzelner WAL-Dateien mit einer anderen Datenbank. Bei übernommenen
Druckaufträgen CUPS und vorhandene Etiketten vor erneutem Drucken prüfen.
Der Git-Checkout kann weiterhin den neueren Commit enthalten; das ist kein Grund,
die Daten zu überschreiben. Die aktive Version über HTTP bzw. `-version` prüfen.

## Offizielle Referenzen

- [Raspberry Pi: Headless-Installation mit Imager](https://www.raspberrypi.com/documentation/computers/getting-started.html)
- [Raspberry Pi: SSH-Zugriff](https://www.raspberrypi.com/documentation/computers/remote-access.html)
- [systemd-Dienst und Neustartverhalten](https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html)
- [Go installieren](https://go.dev/doc/install)
