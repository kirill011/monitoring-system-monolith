package app

import (
	"context"
	"fmt"

	httpbase "net/http"

	"monolith/config"

	"monolith/internal/repo/pg"
	"monolith/internal/services"
	"monolith/internal/transport/http"
	"monolith/pkg/closer"
	"monolith/pkg/logger"
	"monolith/pkg/postgres"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func Run(ctx context.Context, cfg *config.Config, stop context.CancelFunc) {
	log := logger.New(logger.Config{
		LogLevel:    cfg.Logger.LogLevel,
		ServiceName: cfg.Logger.ServiceName,
		LogPath:     cfg.Logger.LogPath,
	})

	postgresDB, err := postgres.NewPostgres(postgres.Config{
		DataSource:        cfg.Postgres.DataSource,
		ApplicationSchema: cfg.Postgres.ApplicationSchema,
	})
	if err != nil {
		log.Fatal("init postgresDB error", zap.Error(err))
	}

	_, err = postgresDB.RunMigrations(cfg.Postgres.PathToMigrations, cfg.Postgres.ApplicationSchema)
	if err != nil {
		log.Fatal("migration error", zap.Error(err))
	}

	postgresDB, err = postgres.NewPostgres(postgres.Config{
		DataSource:        cfg.Postgres.DataSource,
		ApplicationSchema: cfg.Postgres.ApplicationSchema,
	})

	authRepo := pg.NewAuthRepo(postgresDB, log)
	tagsRepo := pg.NewTagsRepo(postgresDB, log)
	devicesRepo := pg.NewDevicesRepo(postgresDB, log)
	messagesRepo := pg.NewMessagesRepo(postgresDB, log)

	authService := services.NewAuthService(authRepo)
	devicesService := services.NewDevicesService(devicesRepo)
	messagesService := services.NewMessagesService(services.MessagesServiceConfig{
		MessageRepo:        messagesRepo,
		TagRepo:            tagsRepo,
		DevicesRepo:        devicesRepo,
		Log:                log,
		NotificationPeriod: cfg.Service.NotificationPeriod,
	})
	tagsService := services.NewTagsService(tagsRepo, messagesService)
	deviceChecker := services.NewDeviceHandler(devicesRepo)

	httpServer := http.NewServer(http.Config{
		Log:             log,
		JwtKey:          cfg.Server.JwtKey,
		Addr:            cfg.Server.Addr,
		LogQuerys:       cfg.Server.LogQuerys,
		AuthHandlers:    authService,
		DevicesHandlers: devicesService,
		TagsHandler:     tagsService,
		ReportsHandler:  messagesService,
		DeviceChecker:   deviceChecker,
	})

	go func() {
		if err := httpServer.Run(); err != nil {
			log.Error(fmt.Sprintf("error occurred while running HTTP server: %v", err))
			stop()
		}
	}()

	go func() {
		httpbase.Handle("/metrics", promhttp.Handler())
		if err := httpbase.ListenAndServe(":9081", nil); err != nil {
			log.Error(fmt.Errorf("error occurred while running http server: %w", err).Error())
			stop()
		}
	}()

	log.Info("start http server", zap.String("listen_on", cfg.Server.Addr))

	// Shutdown
	<-ctx.Done()

	log.Info("Shutdown start")

	closer := closer.Closer{}

	closer.Add(httpServer.Stop)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()

	if err := closer.Close(shutdownCtx); err != nil {
		log.Error("Closer", zap.Error(err))
		return
	}

	log.Info("Shutdown success")
}
