package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin
		return true
	},
}

func startServer() {
	http.HandleFunc("/create_pod", createPodRequest)

	http.HandleFunc("/list_pod", listPodsWebSocket)

	log.Info().Msgf("Starting server on port %s", port)
	http.ListenAndServe(":"+port, nil)
}

func listPodsWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection")
		return
	}
	defer conn.Close()

	log.Info().Msg("Opening websocket to list pods")

	for {
		select {
		case <-r.Context().Done():
			log.Info().Msg("Closing list pods websocket")
			return
		default:
			if podsList != nil {
				pods := []map[string]any{}

				for _, pod := range podsList.Items {
					podOut := map[string]any{
						"podName":           pod.Name,
						"ready":             pod.Status.ContainerStatuses[0].Ready,
						"status":            pod.Status.Phase,
						"restarts":          pod.Status.ContainerStatuses[0].RestartCount,
						"creationTimestamp": pod.CreationTimestamp.Time,
					}
					podOut["expirationTimestamp"], _ = podExpiration(pod)

					pods = append(pods, podOut)
				}
				sendMessage(conn, map[string]any{"pods": pods})
			}

			time.Sleep(1 * time.Second)
		}
	}
}

// func deletePodRequest(w http.ResponseWriter, r *http.Request) {
// 	podName := nameGenerator.GetString()
// 	podSecret := secretGenerator.GetString()

// 	logger := log.With().Str("podName", podName).Logger()
// 	logger.Info().Msg("Client requested pod creation")

// 	data := map[string]any{"podName": podName, "podSecret": podSecret}

// 	if err := createPod(r.Context(), podName, podSecret); err != nil {
// 		logger.Error().Err(err).Msg("Failed to create pod")
// 		sendResponse(w, "Failed to create pod", data)
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	sendResponse(w, "Pod created", data)
// }

func createPodRequest(w http.ResponseWriter, r *http.Request) {
	podName := nameGenerator.GetString()
	podSecret, err := generatePassword(30)
	if err != nil {
		panic(err)
	}

	logger := log.With().Str("podName", podName).Logger()
	logger.Info().Msg("Client requested pod creation")

	data := map[string]any{"podName": podName, "podSecret": podSecret}

	if err := createPod(r.Context(), podName, podSecret); err != nil {
		logger.Error().Err(err).Msg("Failed to create pod")
		sendResponse(w, "Failed to create pod", data)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sendResponse(w, "Pod created", data)
}

func sendResponse(w http.ResponseWriter, message string, data map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"message": message, "data": data})
}

func sendMessage(conn *websocket.Conn, message map[string]any) {
	msg, _ := json.Marshal(message)

	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Error().Err(err).Msg("Failed to write message")
	}
}

func generatePassword(length int) (string, error) {
	// Create a byte slice to hold the random bytes
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode the random bytes to a string and truncate to the desired length
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
