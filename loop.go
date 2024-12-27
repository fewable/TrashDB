package main

import (
	"context"
	"time"
)

func startEventLoop() {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		listRedisPods(ctx)
	}
}
