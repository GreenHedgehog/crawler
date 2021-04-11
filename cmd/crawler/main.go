package main

import (
	"crawler/internal/fetcher"
	"crawler/internal/logger"
	"crawler/internal/service"
	"crawler/internal/storage"
)

func main() {
	defer logger.G().Sync()

	fetcher, err := fetcher.New()
	if err != nil {
		logger.G().Sugar().Fatalf("init fetcher failed: %v", err)
	}

	storage, err := storage.New("mongodb://admin:admin@mongo:27017")
	if err != nil {
		logger.G().Sugar().Fatalf("init fetcher failed: %v", err)
	}

	service, err := service.NewServer(fetcher, storage)
	if err != nil {
		logger.G().Sugar().Fatalf("init server failed: %v", err)
	}

	if err = service.Start(":8080"); err != nil {
		logger.G().Sugar().Errorf("server failed: %v", err)
	}
	logger.G().Sugar().Info("service stop")
}
