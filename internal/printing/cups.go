package printing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/models"
)

type Command func(context.Context, string, []byte, ...string) ([]byte, error)
type Printer struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func systemCommand(ctx context.Context, name string, input []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	cmd.Stdin = bytes.NewReader(input)
	return cmd.CombinedOutput()
}
func USBPrinters(ctx context.Context, run Command) ([]Printer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := run(ctx, "lpstat", nil, "-h", "localhost", "-v")
	if err != nil {
		return nil, errors.New("Lokales CUPS nicht erreichbar. CUPS und Argox-Treiber auf dem Server einrichten.")
	}
	result := []Printer{}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "device for ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "device for "), ": ", 2)
		if len(parts) != 2 || !models.ValidPrinterQueue(parts[0]) || !strings.HasPrefix(strings.TrimSpace(parts[1]), "usb://") {
			continue
		}
		state, err := run(ctx, "lpstat", nil, "-h", "localhost", "-p", parts[0], "-l")
		message := strings.TrimSpace(string(state))
		if err != nil {
			message = "Status derzeit nicht verfügbar"
		}
		if len(message) > 1000 {
			message = message[:1000]
		}
		result = append(result, Printer{Name: parts[0], State: message})
		if len(result) >= 20 {
			break
		}
	}
	return result, nil
}
func checkQueue(ctx context.Context, run Command, queue string) error {
	if !models.ValidPrinterQueue(queue) {
		return errors.New("Ungültige Druckerwarteschlange.")
	}
	printers, err := USBPrinters(ctx, run)
	if err != nil {
		return err
	}
	for _, p := range printers {
		if p.Name == queue {
			if strings.Contains(p.State, " disabled ") || strings.Contains(p.State, "Status derzeit") {
				return errors.New("USB-Drucker angehalten oder Status nicht verfügbar. CUPS prüfen.")
			}
			return nil
		}
	}
	return errors.New("Die gewählte USB-Warteschlange ist in lokalem CUPS nicht vorhanden.")
}

var cupsJob = regexp.MustCompile(`request id is ([A-Za-z0-9_.-]+-[0-9]+)`)

func submit(ctx context.Context, run Command, s models.LabelSettings, id int64, pdf []byte) (string, error) {
	out, err := run(ctx, "lp", pdf, "-h", "localhost", "-d", s.Queue, "-n", fmt.Sprint(s.Copies), "-t", fmt.Sprintf("Reifenapp-Etikett-%d", id), "-o", "media=Custom.78x51mm", "-o", "print-scaling=none", "-o", "orientation-requested=3", "-o", "sides=one-sided", "-o", "job-sheets=none")
	if err != nil {
		return "", errors.New("Druckübergabe nicht bestätigt. Vor erneutem Druck CUPS und Drucker auf bereits vorhandenen Auftrag prüfen.")
	}
	match := cupsJob.FindSubmatch(out)
	if len(match) != 2 {
		return "", errors.New("CUPS-Auftragsnummer nicht eindeutig erkannt. Vor erneutem Druck Warteschlange prüfen.")
	}
	return string(match[1]), nil
}
