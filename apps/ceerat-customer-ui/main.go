package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kaansari/apps-repo/apps/ceerat-customer-ui/internal/apiclient"
	"github.com/kaansari/apps-repo/apps/ceerat-customer-ui/internal/config"
	"github.com/kaansari/apps-repo/apps/ceerat-customer-ui/internal/server"
)

func main() {
	cfg := config.LoadFromEnv()

	client, err := apiclient.New(cfg.APIBaseURL)
	if err != nil {
		log.Fatalf("create api client: %v", err)
	}
	defer client.Close()

	s, err := server.New(cfg, client)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("ceerat-customer-ui listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
