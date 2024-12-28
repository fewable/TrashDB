package trashdb

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/fewable/words"
	"github.com/rs/zerolog/log"

	"github.com/gorilla/websocket"
)

var nameGenerator = words.NewBuilder().WithSeparator("-").AddMediumWord().AddMediumWord()

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin
		return true
	},
}

// TODO: this feels like a hack to get these initialized. Should be a better way
var namespace string

func StartServer(port string, ns string) {
	namespace = ns

	http.HandleFunc("/create_pod", createPodRequest)

	http.HandleFunc("/delete_pod", deletePodRequest)

	http.HandleFunc("/list_pod", listPodWebSocket)

	log.Info().Msgf("Starting server on port %s", port)
	http.ListenAndServe(":"+port, nil)
}

func listPodWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection to websocket")
		return
	}
	defer conn.Close()

	for {
		select {
		case <-r.Context().Done():
			return
		default:
			sendMessage(conn, "Got pods", map[string]any{"pods": podsCache})

			time.Sleep(1 * time.Second)
		}
	}
}

func deletePodRequest(w http.ResponseWriter, r *http.Request) {
	type deletePodRequest struct {
		PodName   string `json:"podName"`
		PodSecret string `json:"podSecret"`
	}

	var body deletePodRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	data := map[string]any{"podName": body.PodName}

	err := DeletePodWithSecret(r.Context(), nil, namespace, body.PodName, body.PodSecret)
	if err != nil {
		sendResponse(w, http.StatusBadRequest, err.Error(), data)
		return
	}

	sendResponse(w, http.StatusOK, "Pod deleted", data)
}

func createPodRequest(w http.ResponseWriter, r *http.Request) {
	type createPodRequest struct {
		PodName  string `json:"podName"`
		Duration int    `json:"duration"`
	}

	var body createPodRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	podName := body.PodName
	if podName == "" {
		podName = nameGenerator.GetString()
	}
	podSecret := generatePassword(30)

	duration := time.Duration(body.Duration) * time.Minute
	if body.Duration == 0 {
		// TODO: allow setting this lower for tophatting
		duration = 10 * time.Minute
	}

	data := map[string]any{"podName": podName, "podSecret": podSecret}

	if pod, err := CreatePod(r.Context(), nil, namespace, podName, podSecret, duration); err != nil {
		sendResponse(w, http.StatusBadRequest, err.Error(), data)
		return
	} else {
		data["app.trashdb/expiration"] = pod.Annotations["app.trashdb/expiration"]
	}

	sendResponse(w, http.StatusOK, "Pod created", data)
}

func generatePassword(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

func sendResponse(w http.ResponseWriter, status int, message string, data map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]any{
		"message": message,
		"data":    data,
	}

	// Handle encoding errors explicitly
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, `{"message": "Internal Server Error"}`, http.StatusInternalServerError)
	}
}

func sendMessage(conn *websocket.Conn, message string, data map[string]any) {
	response := map[string]any{
		"message": message,
		"data":    data,
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error().Err(err).Msg("Failed to send message over websocket")
	}
}
