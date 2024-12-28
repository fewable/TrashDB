package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/taimoorgit/trashdb/trashdb"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var client *kubernetes.Clientset

func env(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return value
}

func initKubernetesClient() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}

func startEventLoop(namespace string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		trashdb.ListPods(ctx, nil, namespace)

		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer deleteCancel()
		trashdb.DeleteExpiredPods(deleteCtx, namespace)
	}
}

func main() {
	namespace := env("NAMESPACE", "trashdb")
	port := env("PORT", "8080")

	c := initKubernetesClient()
	trashdb.SetClient(c)

	go startEventLoop(namespace)

	go trashdb.StartServer(port, namespace)

	select {}
}
