package printing

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/models"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

func TestCUPSWorkerAndUncertainSubmission(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{true: "uncertain", false: "submitted"}[failed], func(t *testing.T) {
			ctx := context.Background()
			db, err := database.Open(filepath.Join(t.TempDir(), "printer.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			s := models.DefaultLabelSettings()
			s.Queue = "Argox_USB"
			s.Copies = 4
			id, err := database.TestLabelJob(ctx, db, s, receipts.ExampleReceipt("storage"))
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			run := func(ctx context.Context, name string, input []byte, args ...string) ([]byte, error) {
				joined := strings.Join(args, " ")
				if !strings.Contains(joined, "-h localhost") {
					t.Fatal("nonlocal cups")
				}
				if name == "lpstat" {
					if strings.Contains(joined, "-v") {
						return []byte("device for Argox_USB: usb://Argox/Test\ndevice for Network: ipp://example.test\n"), nil
					}
					return []byte("printer Argox_USB is idle. enabled since today"), nil
				}
				if name != "lp" {
					t.Fatal(name)
				}
				calls++
				if !bytes.HasPrefix(input, []byte("%PDF-")) || !strings.Contains(joined, "media=Custom.78x51mm") || !strings.Contains(joined, "-n 4") || !strings.Contains(joined, "print-scaling=none") {
					t.Fatal("wrong print job", joined)
				}
				if failed {
					return nil, errors.New("simulated timeout")
				}
				return []byte("request id is Argox_USB-42 (1 file(s))"), nil
			}
			service := &Service{DB: db, Run: run}
			if err = service.ProcessOne(ctx); err != nil {
				t.Fatal(err)
			}
			j, err := database.LabelJobStatus(ctx, db, id)
			expected := "submitted"
			if failed {
				expected = "uncertain"
			}
			if err != nil || j.Status != expected {
				t.Fatal(j, err)
			}
			if err = service.ProcessOne(ctx); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("unexpected automatic duplicate")
			}
		})
	}
}
func TestMissingPrinterDoesNotSubmit(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "missing.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := models.DefaultLabelSettings()
	s.Queue = "Missing_USB"
	id, err := database.TestLabelJob(ctx, db, s, receipts.ExampleReceipt("storage"))
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{DB: db, Run: func(_ context.Context, name string, _ []byte, _ ...string) ([]byte, error) {
		if name == "lp" {
			t.Fatal("printed without USB queue")
		}
		return nil, errors.New("cups unavailable")
	}}
	if err = service.ProcessOne(ctx); err != nil {
		t.Fatal(err)
	}
	j, err := database.LabelJobStatus(ctx, db, id)
	if err != nil || j.Status != "failed" {
		t.Fatal(j, err)
	}
}

func TestStopWaitsForInFlightPrintStatus(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "stop.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	settings := models.DefaultLabelSettings()
	settings.Queue = "Argox_USB"
	id, err := database.TestLabelJob(ctx, db, settings, receipts.ExampleReceipt("storage"))
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	service := &Service{DB: db, Run: func(ctx context.Context, name string, _ []byte, args ...string) ([]byte, error) {
		if name == "lpstat" {
			if strings.Contains(strings.Join(args, " "), "-v") {
				return []byte("device for Argox_USB: usb://Argox/Test\n"), nil
			}
			return []byte("printer Argox_USB is idle. enabled since today"), nil
		}
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	stop, err := service.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not start")
	}
	stop()
	job, err := database.LabelJobStatus(ctx, db, id)
	if err != nil || job.Status != "uncertain" {
		t.Fatal("stop returned before persistent status", job, err)
	}
}
