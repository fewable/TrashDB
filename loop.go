package main

import (
	"context"
	"time"
)

func startEventLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		listPods(ctx)

		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer deleteCancel()
		deleteExpiredPods(deleteCtx)
	}
}
