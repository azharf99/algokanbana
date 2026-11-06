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
	_ "time/tzdata"

	"algokanbana"
	"algokanbana/internal/api"
	"algokanbana/internal/config"
	"algokanbana/internal/seed"
	"algokanbana/internal/store"
)

func main() {
	cfg := config.Load()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("gagal membuka database SQLite (%s): %v", cfg.DBPath, err)
	}
	defer func() { _ = st.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := seed.EnsureSeeded(ctx, st, cfg.AppMode); err != nil {
		log.Printf("gagal menginisialisasi data awal: %v", err)
		return
	}

	srv := api.NewServer(cfg, st, algokanbana.WebDistFS())
	srv.StartBackground(ctx)

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0, // 0 supports long-lived SSE connections cleanly
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("⚡ AlgoKanbana aktif di http://localhost:%s [mode=%s | safe_readonly_upstream=%v]", cfg.Port, cfg.AppMode, cfg.SafeReadonlyUpstream)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server HTTP berhenti: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Menutup server AlgoKanbana secara aman...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
}
