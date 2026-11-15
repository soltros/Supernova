package main

import (
	"context"
	"fmt"
	"github.com/soltros/Supernova/internal/database"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage: supernova-backup SOURCE_DB NEW_BACKUP_DB")
		os.Exit(2)
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 10*time.Minute)
	defer cancel()
	if err := database.Backup(ctx, os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Verified database backup created:", os.Args[2])
}
