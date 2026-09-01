package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/diosyahrizal/finance-recap-api/internal/importer"
	"github.com/diosyahrizal/finance-recap-api/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	log.Println("Connected to the database successfully")

	recapStore := postgres.NewRecapStore(db)
	importStore := postgres.NewImportJobStore(db)
	fileStore := importer.NewLocalFileStore("./uploads")

	app := &application{
		store:         recapStore,
		importCreator: importStore,
		fileStore:     fileStore,
	}

	log.Fatal(http.ListenAndServe(":8080", app.routes()))
}
