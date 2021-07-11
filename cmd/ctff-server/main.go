package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	api "github.com/nwillems/ctff-server/pkg/service-api"
	"github.com/nwillems/ctff-server/pkg/storage"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	listenAddr     string
	kubeconfigPath string
	logger         = log.New(os.Stdout, "http: ", log.LstdFlags)
)

func getKubeConfig() (*rest.Config, error) {
	if len(kubeconfigPath) == 0 {
		return rest.InClusterConfig()
	} else {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
}

func getStorageBackend(useConfigmapStorage bool, namespace string) (storage.FeatureFlagStore, error) {
	if useConfigmapStorage {
		kubeconfig, err := getKubeConfig()
		if err != nil {
			return nil, err
		}
		store := storage.NewConfigMapBackend(namespace, kubeconfig)
		return store, nil
	} else {
		return storage.NewInMemory(), nil
	}
}

// Would still be awesome to do!
func middlewareServiceAccountAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		clientId := r.Header.Get("Authentication")
		if len(clientId) == 0 {
			//Fail
		}

		config, err := getKubeConfig()
		clientset, err := kubernetes.NewForConfig(config)

		tr := authv1.TokenReview{
			Spec: authv1.TokenReviewSpec{
				Token:     clientId,
				Audiences: []string{"api"},
			},
		}
		result, err := clientset.AuthenticationV1().TokenReviews().Create(context.TODO(), &tr, metav1.CreateOptions{})
		if err != nil {
			//TODO Error handle
		}

		if result.Status.Authenticated {
			next.ServeHTTP(rw, r)
		} else {
			// TODO: Error handle
		}
	}
}

func indexHandleFunc() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "JSON DATA GOES HERE")
	}
}

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":9000", "server listen address")
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Kubeconfig to use, otherwise assume running in-cluster")
	useConfigmapStorage := flag.Bool("use-configmap-storage", true, "Set wether to use the in-memory or configmap(default) storage")
	namespace := flag.String("namespace", "", "The namespace to use for the configmap backend")
	flag.Parse()

	logger.Println("Server is starting...")

	storagebackend, err := getStorageBackend(*useConfigmapStorage, *namespace)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	server := api.NewFlaggerServer(storagebackend, *namespace)
	router.HandleFunc("/", indexHandleFunc())
	// router.Handle("/healthz", healthz())

	// router.HandleFunc("/api/{identity}/flags/{flag_name}", middlewareServiceAccountAuthentication(server.GetFeatureFlagStateHandler)).Methods("GET")
	router.HandleFunc("/api/{identity}/flags/{flag_name}", server.GetFeatureFlagStateHandler).Methods("GET")
	router.HandleFunc("/api/{identity}/register", server.RegisterFeatureFlagsHandler).Methods("POST")
	router.HandleFunc("/api/{identity}/flags", server.ListAllFeatureFlagsHandler).Methods("GET")
	router.HandleFunc("/api/{identity}/flags/{flag_name}", server.SetFeatureFlagStateHandler).Methods("POST")

	http_server := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	if err := http_server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}
}
