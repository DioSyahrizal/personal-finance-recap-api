package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

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

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY is not set")
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

	itemStore := postgres.NewItemStore(db)
	openAIModel := os.Getenv("OPENAI_MODEL")
	if openAIModel == "" {
		openAIModel = "gpt-5.2"
	}

	parser := importer.NewOpenAIParser(openAIModel)

	processor := importer.NewJobProcessor(
		parser,
		itemStore,
		fileStore,
	)

	worker := importer.NewWorker(
		importStore,
		processor,
		5*time.Second,
	)

	go worker.Run(ctx)

	app := &application{
		store:         recapStore,
		importCreator: importStore,
		fileStore:     fileStore,
	}

	log.Fatal(http.ListenAndServe(":8080", app.routes()))
}
