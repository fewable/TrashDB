package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin
		return true
	},
}

func startServer() {
	http.HandleFunc("/api", handleWebSocket)

	log.Info().Msgf("Starting server on port %s", port)
	http.ListenAndServe(":"+port, nil)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection")
		return
	}
	defer conn.Close()

	log.Info().Msg("Client connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Failed to read message")
			break
		}

		switch message := string(msg); message {
		case "create_pod":
			podName := nameGenerator.GetString()
			podSecret := secretGenerator.GetString()

			logger := log.With().Str("podName", podName).Logger()
			logger.Info().Msg("Client requested pod creation")

			sendMessage(conn, map[string]string{"message": "Creating pod", "podName": podName})

			if err := createRedisPod(r.Context(), podName); err != nil {
				sendMessage(conn, map[string]string{"message": "Failed to create pod", "podName": podName})
				continue
			}

			sendMessage(conn, map[string]string{"message": "Pod created: " + podName, "name": podName, "secret": podSecret})
		case "list_pod":
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			if err := listRedisPods(ctx); err != nil {
				sendMessage(conn, map[string]string{"message": "Failed to list pods"})
				continue
			}
		default:
			sendMessage(conn, map[string]string{"message": "Unknown command: " + message})
		}
	}
}

func sendMessage(conn *websocket.Conn, message map[string]string) {
	msg, _ := json.Marshal(message)

	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Error().Err(err).Msg("Failed to write message")
	}
}
