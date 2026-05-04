package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/agent"
	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/config"
	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/httpapi"
	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/platform"
)

func main() {
	cfg := config.Load()

	platformClient, err := platform.New(cfg.UserServiceAddress)
	if err != nil {
		log.Fatalf("connect platform gRPC service: %v", err)
	}
	defer platformClient.Close()

	llm := agent.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	ceeratAgent := agent.New(llm, &agent.ToolRunner{Platform: platformClient})

	api := &httpapi.Server{
		Agent:    ceeratAgent,
		Platform: platformClient,
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("ceerat-agent-service listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down ceerat-agent-service")
}
