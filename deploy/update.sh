#!/usr/bin/env bash
# Run as the SSH/deployment user, not root. Main is parsed before git updates this file.
set -Eeuo pipefail
umask 077

main() {
    if [[ $EUID -eq 0 ]]; then
        echo "Als normalen SSH-Benutzer mit sudo-Recht starten, nicht als root." >&2
        return 1
    fi
    source_dir=${REIFENAPP_SOURCE_DIR:-/opt/reifenapp/source}
    bin_dir=${REIFENAPP_BIN_DIR:-/opt/reifenapp/bin}
    state_dir=${REIFENAPP_STATE_DIR:-/var/lib/reifenapp}
    backup_root=${REIFENAPP_BACKUP_DIR:-/var/backups/reifenapp}
    service_user=${REIFENAPP_SERVICE_USER:-reifenapp}
    service_name=${REIFENAPP_SERVICE:-reifenapp.service}
    health_url=${REIFENAPP_HEALTH_URL:-http://127.0.0.1:8080/api/health}
    expected_remote=https://github.com/karst3nl/reifenapp-2.git
    stage=''
    target=''
    stamp=''
    backup_dir=''
    stopped=false
    migration_started=false
    success=false
    for command in git go gcc sudo systemctl sqlite3 curl python3 flock; do
        command -v "$command" >/dev/null || { echo "Benötigter Befehl fehlt: $command" >&2; return 1; }
    done
    cd "$source_dir"
    [[ $(git remote get-url origin) == "$expected_remote" ]] || { echo "Abbruch: origin ist nicht ReifenApp-2." >&2; return 1; }
    [[ $(git branch --show-current) == beta ]] || { echo "Abbruch: Für dieses Verfahren muss beta ausgecheckt sein." >&2; return 1; }
    [[ -z $(git status --porcelain) ]] || { echo "Abbruch: Lokale Codeänderungen zuerst sichern und klären." >&2; return 1; }
    exec 9>"$(git rev-parse --git-path reifenapp-update.lock)"
    flock -n 9 || { echo "Ein Update läuft bereits." >&2; return 1; }
    sudo -v
    sudo test -f "$state_dir/data/reifenapp.sqlite" || { echo "Keine bestehende Produktionsdatenbank. Zuerst die Ersteinrichtung durchführen." >&2; return 1; }
    [[ -x "$bin_dir/reifenapp" ]] || { echo "Installierte Binärdatei fehlt." >&2; return 1; }
    git fetch origin beta
    git merge-base --is-ancestor HEAD origin/beta || { echo "Abbruch: Lokale Commits weichen von origin/beta ab. Nicht automatisch zurücksetzen." >&2; return 1; }
    git merge --ff-only origin/beta
    target=$(git rev-parse HEAD)
    stage=$(mktemp -d)
    stamp=$(date -u +%Y%m%dT%H%M%S)-$$
    backup_dir="$backup_root/$stamp"
    # State is global so EXIT cleanup also works after main returns on an error.
    cleanup() {
        local status=$?
        trap - EXIT
        if [[ $success != true && $stopped == true ]]; then
            if [[ $migration_started == false ]]; then
                echo "Update vor Migration abgebrochen; bisherige Version wird wieder gestartet." >&2
                sudo systemctl start "$service_name" || true
            else
                sudo systemctl stop "$service_name" || true
                echo "Update nach Beginn der Migration abgebrochen. Daten bleiben erhalten; Dienst bleibt gestoppt." >&2
                echo "Sicherung/alte Binärdatei: $backup_dir. Anleitung: docs/raspberry-pi.md (Fehler/Rückfall)." >&2
            fi
        fi
        rm -rf -- "$stage"
        exit "$status"
    }
    trap cleanup EXIT
    export CGO_ENABLED=1 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
    echo "Prüfe und baue Commit $target; der bisherige Dienst läuft weiter."
    go test ./...
    go vet ./...
    go build -trimpath -ldflags "-X main.version=$target" -o "$stage/reifenapp" .
    [[ $("$stage/reifenapp" -version) == "$target" ]] || { echo "Build-Version stimmt nicht." >&2; return 1; }
    sudo -v
    echo "Stoppe Dienst für Sicherung und Migration."
    sudo systemctl stop "$service_name"
    stopped=true
    sudo install -d -m 0700 "$backup_root" "$backup_dir"
    sudo install -m 0755 "$bin_dir/reifenapp" "$backup_dir/reifenapp.previous"
    # SQLite backup API includes committed WAL data; do not copy an active database file.
    # Timestamp and fixed standard paths contain no single quotes. Reject other paths.
    [[ $backup_dir != *"'"* ]] || { echo "Unzulässiger Sicherungspfad." >&2; return 1; }
    sudo sqlite3 -readonly "$state_dir/data/reifenapp.sqlite" ".backup '$backup_dir/reifenapp.sqlite'"
    [[ $(sudo sqlite3 -readonly "$backup_dir/reifenapp.sqlite" 'PRAGMA integrity_check;') == ok ]] || { echo "Sicherungsprüfung fehlgeschlagen." >&2; return 1; }
    if sudo test -d "$state_dir/local"; then
        sudo cp -a "$state_dir/local" "$backup_dir/local"
    fi
    sudo install -m 0600 /dev/null "$backup_dir/commit.txt"
    printf '%s\n' "$target" | sudo tee "$backup_dir/commit.txt" >/dev/null
    sudo chmod 0600 "$backup_dir/commit.txt" "$backup_dir/reifenapp.sqlite"
    sudo install -m 0755 "$stage/reifenapp" "$bin_dir/reifenapp.next"
    migration_started=true
    sudo -u "$service_user" -- bash -c 'cd "$1" && exec "$2" -migrate-only' bash "$state_dir" "$bin_dir/reifenapp.next"
    sudo mv -T "$bin_dir/reifenapp.next" "$bin_dir/reifenapp"
    sudo systemctl start "$service_name"
    local healthy=false
    for ((attempt=0; attempt<30; attempt++)); do
        if sudo systemctl is-active --quiet "$service_name" &&
            curl --fail --silent --max-time 2 "$health_url" |
                python3 -c 'import json,sys; d=json.load(sys.stdin); sys.exit(0 if d.get("status")=="ok" and d.get("version")==sys.argv[1] else 1)' "$target" 2>/dev/null; then
            healthy=true
            break
        fi
        sleep 1
    done
    [[ $healthy == true ]] || { echo "Dienst-/HTTP-Prüfung fehlgeschlagen. journalctl -u $service_name prüfen." >&2; return 1; }
    success=true
    echo "Update erfolgreich: $target"
    echo "Sicherung und bisherige Binärdatei: $backup_dir"
    echo "systemd-Unit und /etc/reifenapp wurden nicht automatisch überschrieben."
}

main "$@"; exit
