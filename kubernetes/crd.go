package kubernetes

import (
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"

	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//GetK8sClientConfig creates a Kubernetes client config
//This is a duplicate from Helm
func GetK8sClientConfig(kubeConfig []byte) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig != nil {
		apiconfig, err := clientcmd.Load(kubeConfig)
		if err != nil {
			return nil, err
		}

		clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
		log.Debug("Use K8S RemoteCluster Config: ", config.ServerName)
	} else {
		return nil, errors.New("kubeconfig value is nil")
	}
	if err != nil {
		return nil, fmt.Errorf("create kubernetes config failed: %v", err)
	}
	return config, nil
}

// GetApiExtensionClient helper
func GetApiExtensionClient(kubeConfig []byte) (*apiextcs.Clientset, error) {
	config, err := GetK8sClientConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	clientset, err := apiextcs.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset, nil
}
