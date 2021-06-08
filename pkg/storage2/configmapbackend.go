package pkg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	restclient "k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/retry"
)

type ConfigMapBackend struct {
	namespace string
}

func NewConfigMapBackend(namespace string) *ConfigMapBackend {

	return &ConfigMapBackend{
		namespace: namespace,
	}
}

func getConfigForAuth(authentication_identity string) (*restclient.Config, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, errors.New("Environment variables not set(KUBERNETES_SERVICE_HOST, KUBERNETES_SERVICE_PORT)")
	}

	tlsClientConfig := restclient.TLSClientConfig{}
	rootCAFile := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	if _, err := certutil.NewPool(rootCAFile); err != nil {
		log.Printf("[ERROR] Expected to load root CA config from %s, but got err: %v", rootCAFile, err)
	} else {
		tlsClientConfig.CAFile = rootCAFile
	}

	return &restclient.Config{
		// TODO: switch to using cluster DNS.
		Host:            "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: tlsClientConfig,
		BearerToken:     authentication_identity,
	}, nil
}

func (db *ConfigMapBackend) getConfigMap(config *restclient.Config, identity string) (*v1.ConfigMap, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	api := clientset.CoreV1()

	configmapName := identity // do some sanity check, cleanup/sanitize
	getOpts := metav1.GetOptions{}

	return api.ConfigMaps(db.namespace).Get(context.TODO(), configmapName, getOpts)
}

func (db *ConfigMapBackend) updateConfigMap(config *restclient.Config, identity string, newData map[string]string) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	api := clientset.CoreV1().ConfigMaps(db.namespace)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		configmapName := identity
		result, getErr := api.Get(context.TODO(), configmapName, metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get latest version of Configmap: %v", getErr))
		}

		result.Data = newData
		_, updateErr := api.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})
}

func (db *ConfigMapBackend) GetFeatureFlagState(authentication_identity string, identity, flag_name, flag_context_id string) (bool, error) {
	config, err := getConfigForAuth(authentication_identity)
	if err != nil {
		return false, err
	}

	configMap, err := db.getConfigMap(config, identity)
	if err != nil {
		return false, err
	}
	featureFlags := configMap.Data

	flagValue, found := featureFlags[flag_name] // consider the flag value should be a json, and if it doesn't parse as such, try bool
	if found {
		return strconv.ParseBool(flagValue)
	}
	return false, fmt.Errorf("No flag exist with that name.")
}

func (db *ConfigMapBackend) RegisterServiceFlags(authentication_identity string, identity string, flags []string) error {
	// get current configmap
	config, err := getConfigForAuth(authentication_identity)
	if err != nil {
		return err
	}
	configMap, err := db.getConfigMap(config, identity)
	if err != nil {
		return err
	}
	currentFeatureFlags := configMap.Data

	// diff new and current list of feature flags
	newConfigMap := make(map[string]string, len(flags))
	for _, flag := range flags {
		newConfigMap[flag] = strconv.FormatBool(false)

		currentValue, found := currentFeatureFlags[flag]
		if found {
			newConfigMap[flag] = currentValue
		}
	}

	// update configmap
	return db.updateConfigMap(config, identity, newConfigMap)
}
