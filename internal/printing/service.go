package printing

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/labels"
)

type Service struct {
	DB  *sql.DB
	Run Command
}

func New(db *sql.DB) *Service                                      { return &Service{DB: db, Run: systemCommand} }
func (s *Service) Printers(ctx context.Context) ([]Printer, error) { return USBPrinters(ctx, s.Run) }
func (s *Service) Check(ctx context.Context, queue string) error {
	return checkQueue(ctx, s.Run, queue)
}
func (s *Service) Start(ctx context.Context) (context.CancelFunc, error) {
	if err := database.RecoverLabelJobs(ctx, s.DB); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.ProcessOne(ctx); err != nil && !errors.Is(err, context.Canceled) {
					log.Print("Etikettendruck: Auftragsstatus konnte nicht aktualisiert werden; Administration prüfen.")
				}
			}
		}
	}()
	return cancel, nil
}
func (s *Service) ProcessOne(ctx context.Context) error {
	j, err := database.ClaimLabelJob(ctx, s.DB)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	status, cupsID, message := "failed", "", ""
	pdf, err := labels.PDF(j.Settings, j.Receipt)
	if err != nil {
		message = "Etikett konnte nicht erstellt werden. Vorlage und Feldgrößen prüfen."
	} else if err = s.Check(ctx, j.Settings.Queue); err != nil {
		message = err.Error()
	} else {
		cupsID, err = submit(ctx, s.Run, j.Settings, j.ID, pdf)
		if err != nil {
			status = "uncertain"
			message = err.Error()
		} else {
			status = "submitted"
			message = "An CUPS übergeben. Dies bestätigt noch nicht den physischen Ausdruck."
		}
	}
	// Persist even when a browser disconnects or shutdown cancels the command.
	finishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return database.FinishLabelJob(finishCtx, s.DB, j.ID, status, cupsID, message)
}
