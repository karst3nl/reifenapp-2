package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/karst3nl/reifenapp-2/internal/database"
	"github.com/karst3nl/reifenapp-2/internal/handlers"
	"github.com/karst3nl/reifenapp-2/internal/receipts"
)

//go:embed web
var assets embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "Listen address")
	dev := flag.Bool("dev", false, "Use separate fictional development database")
	upgrade := flag.Bool("migrate", false, "Upgrade existing database after automatic backup (stop other servers first)")
	headerPath := flag.String("receipt-config", "local/receipt-header.json", "Optional local workshop header JSON")
	flag.Parse()
	header, headerErr := receipts.LoadHeader(*headerPath)
	if headerErr != nil {
		log.Fatal(headerErr)
	}
	path := "data/reifenapp.sqlite"
	if *dev {
		path = "data/development.sqlite"
	}
	_, statErr := os.Stat(path)
	fresh := os.IsNotExist(statErr)
	db, err := database.OpenWithMigration(path, *upgrade)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if *dev && fresh {
		if err = database.SeedDevelopment(db); err != nil {
			log.Fatal(err)
		}
	}
	web, _ := fs.Sub(assets, "web")
	mux := http.NewServeMux()
	admin, adminErr := handlers.NewAdmin(db, header)
	if adminErr != nil {
		log.Fatal(adminErr)
	}
	admin.Register(mux)
	mux.HandleFunc("GET /api/search", handlers.Search(db))
	mux.HandleFunc("GET /api/history", handlers.History(db))
	mux.HandleFunc("GET /api/warnings", handlers.TreadWarnings(db))
	mux.HandleFunc("GET /api/season", handlers.SeasonProgress(db))
	mux.HandleFunc("GET /api/inactivity", handlers.Inactivity(db))
	mux.HandleFunc("GET /api/receipts/{claim}/{operation}", handlers.Receipts(db, header))
	mux.HandleFunc("GET /api/customers", handlers.Customers(db))
	mux.HandleFunc("POST /api/storage", handlers.Storage(db))
	mux.HandleFunc("POST /api/releases/check", handlers.CheckRelease(db))
	mux.HandleFunc("POST /api/releases", handlers.Releases(db))
	mux.HandleFunc("POST /api/wheel-changes", handlers.WheelChanges(db))
	mux.HandleFunc("POST /api/reservations", handlers.Reservations(db))
	mux.HandleFunc("POST /api/storage/check", handlers.CheckStorage(db))
	mux.HandleFunc("GET /api/storage/options", handlers.StorageOptions(db))
	mux.Handle("GET /", http.FileServer(http.FS(web)))
	server := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("Reifenapp: http://%s (development=%t)", *addr, *dev)
	log.Fatal(server.ListenAndServe())
}
