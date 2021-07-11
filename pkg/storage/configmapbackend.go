package storage

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type ConfigMapBackend struct {
	namespace  string
	kubeconfig *restclient.Config
}

func parse_flag_state(value string) bool {
	//TODO: Make parsing of the thing, such that we can support contexts
	result, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return result
}

func NewConfigMapBackend(namespace string, kubeconfig *restclient.Config) *ConfigMapBackend {

	return &ConfigMapBackend{
		namespace:  namespace,
		kubeconfig: kubeconfig,
	}
}

func (db *ConfigMapBackend) getConfigMap(identity string) (*v1.ConfigMap, error) {
	clientset, err := kubernetes.NewForConfig(db.kubeconfig)
	if err != nil {
		return nil, err
	}

	api := clientset.CoreV1()

	configmapName := identity // do some sanity check, cleanup/sanitize
	getOpts := metav1.GetOptions{}

	return api.ConfigMaps(db.namespace).Get(context.TODO(), configmapName, getOpts)
}

func (db *ConfigMapBackend) updateConfigMap(identity string, newData map[string]string) error {
	clientset, err := kubernetes.NewForConfig(db.kubeconfig)
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

func (db *ConfigMapBackend) GetFeatureFlagState(authentication_identity string, identity, flag_name string) (*FeatureFlag, error) {
	configMap, err := db.getConfigMap(identity)
	if err != nil {
		return &FeatureFlag{Name: flag_name, State: false}, err
	}
	featureFlags := configMap.Data

	flagValue, found := featureFlags[flag_name] // consider the flag value should be a json, and if it doesn't parse as such, try bool
	if found {
		value, err := strconv.ParseBool(flagValue)
		if err != nil {
			return &FeatureFlag{Name: flag_name, State: false}, err
		}

		return &FeatureFlag{Name: flag_name, State: value}, nil
	}
	return &FeatureFlag{Name: flag_name, State: false}, fmt.Errorf("No flag exist with that name.")
}

func (db *ConfigMapBackend) RegisterFeatureFlags(authentication_identity string, identity string, flags []*FeatureFlag) error {
	// get current configmap
	configMap, err := db.getConfigMap(identity)
	if err != nil {
		return err
	}
	currentFeatureFlags := configMap.Data

	// diff new and current list of feature flags
	newConfigMap := make(map[string]string, len(flags))
	for _, flag := range flags {
		newConfigMap[flag.Name] = strconv.FormatBool(false)

		currentValue, found := currentFeatureFlags[flag.Name]
		if found {
			newConfigMap[flag.Name] = currentValue
		}
	}

	// update configmap
	return db.updateConfigMap(identity, newConfigMap)
}

func (db *ConfigMapBackend) GetAllFeatureFlags(authentication_identityt, identity string) ([]*FeatureFlag, error) {
	configMap, err := db.getConfigMap(identity)
	if err != nil {
		return nil, err
	}

	currentFeatureFlags := configMap.Data
	var result []*FeatureFlag
	for key, val := range currentFeatureFlags {
		parsed_value := parse_flag_state(val)
		flag := FeatureFlag{
			Name:  key,
			State: parsed_value,
		}

		result = append(result, &flag)
	}
	return result, nil
}

func (db *ConfigMapBackend) SetFeatureFlagState(authentication_id string, identity string, flag_name string, flag_state string) error {
	configMap, err := db.getConfigMap(identity)
	if err != nil {
		return err
	}

	currentFeatureFlags := configMap.Data
	if _, ok := currentFeatureFlags[flag_name]; !ok {
		return fmt.Errorf("No flag with name exists: %v", flag_name)
	}

	_, err = strconv.ParseBool(flag_state)
	if err != nil {
		return err
	}

	currentFeatureFlags[flag_name] = flag_state
	return db.updateConfigMap(identity, currentFeatureFlags)
}
