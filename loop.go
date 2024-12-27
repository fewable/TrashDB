package main

import (
	"context"
	"time"
)

func startEventLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		listRedisPods(ctx)
	}
}
