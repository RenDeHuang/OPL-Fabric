package kubeconfig

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
)

func LoadRESTConfig(path string) (*rest.Config, error) {
	if path == "" {
		path = os.Getenv("KUBECONFIG")
	}
	if path == "" {
		return rest.InClusterConfig()
	}
	return restConfigFromKubeconfig(path)
}

func restConfigFromKubeconfig(path string) (*rest.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var kubeconfig kubeconfigFile
	if err := yaml.Unmarshal(data, &kubeconfig); err != nil {
		return nil, err
	}
	contextName := kubeconfig.CurrentContext
	if contextName == "" && len(kubeconfig.Contexts) > 0 {
		contextName = kubeconfig.Contexts[0].Name
	}
	contextEntry, ok := kubeconfig.contextByName(contextName)
	if !ok {
		return nil, fmt.Errorf("kubeconfig context %q not found", contextName)
	}
	clusterEntry, ok := kubeconfig.clusterByName(contextEntry.Context.Cluster)
	if !ok {
		return nil, fmt.Errorf("kubeconfig cluster %q not found", contextEntry.Context.Cluster)
	}
	userEntry, ok := kubeconfig.userByName(contextEntry.Context.User)
	if !ok {
		return nil, fmt.Errorf("kubeconfig user %q not found", contextEntry.Context.User)
	}
	if clusterEntry.Cluster.Server == "" {
		return nil, errors.New("kubeconfig cluster server is required")
	}
	return &rest.Config{
		Host:        clusterEntry.Cluster.Server,
		BearerToken: userEntry.User.Token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: clusterEntry.Cluster.InsecureSkipTLSVerify,
			CAData:   decodeConfigData(clusterEntry.Cluster.CertificateAuthorityData),
			CertData: decodeConfigData(userEntry.User.ClientCertificateData),
			KeyData:  decodeConfigData(userEntry.User.ClientKeyData),
		},
	}, nil
}

type kubeconfigFile struct {
	CurrentContext string                `yaml:"current-context"`
	Clusters       []kubeconfigCluster   `yaml:"clusters"`
	Contexts       []kubeconfigContext   `yaml:"contexts"`
	Users          []kubeconfigNamedUser `yaml:"users"`
}

type kubeconfigCluster struct {
	Name    string `yaml:"name"`
	Cluster struct {
		Server                   string `yaml:"server"`
		CertificateAuthorityData string `yaml:"certificate-authority-data"`
		InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
	} `yaml:"cluster"`
}

type kubeconfigContext struct {
	Name    string `yaml:"name"`
	Context struct {
		Cluster string `yaml:"cluster"`
		User    string `yaml:"user"`
	} `yaml:"context"`
}

type kubeconfigNamedUser struct {
	Name string `yaml:"name"`
	User struct {
		Token                 string `yaml:"token"`
		ClientCertificateData string `yaml:"client-certificate-data"`
		ClientKeyData         string `yaml:"client-key-data"`
	} `yaml:"user"`
}

func (k kubeconfigFile) contextByName(name string) (kubeconfigContext, bool) {
	for _, entry := range k.Contexts {
		if entry.Name == name {
			return entry, true
		}
	}
	return kubeconfigContext{}, false
}

func (k kubeconfigFile) clusterByName(name string) (kubeconfigCluster, bool) {
	for _, entry := range k.Clusters {
		if entry.Name == name {
			return entry, true
		}
	}
	return kubeconfigCluster{}, false
}

func (k kubeconfigFile) userByName(name string) (kubeconfigNamedUser, bool) {
	for _, entry := range k.Users {
		if entry.Name == name {
			return entry, true
		}
	}
	return kubeconfigNamedUser{}, false
}

func decodeConfigData(value string) []byte {
	if value == "" {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return []byte(value)
	}
	return decoded
}
