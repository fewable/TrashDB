package main

import (
	"os"

	"github.com/fewable/words"

	"k8s.io/client-go/kubernetes"
)

var client *kubernetes.Clientset

var nameGenerator = words.NewBuilder().WithSeparator("-").AddMediumWord().AddMediumWord()

var secretGenerator = words.NewBuilder().WithSecureRandomness().WithSeparator("-").AddMediumWord().AddLongWord()

var namespace string

var port string

func env(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}

func main() {
	namespace = env("NAMESPACE", "trashdb")
	port = env("PORT", "8080")

	initKubernetesClient()

	go startServer()

	go startEventLoop()

	select {}
}
