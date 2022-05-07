package config

// import (
// 	"context"
// 	"fmt"

// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/rest"
// 	"k8s.io/client-go/tools/clientcmd"
// )

// type Option func(*options)

// type options struct {
// 	Namespace  string
// 	KubeConfig string
// 	Master     string
// }

// func Namespace(ns string) Option {
// 	return func(o *options) {
// 		o.Namespace = ns
// 	}
// }

// func KubeConfig(config string) Option {
// 	return func(o *options) {
// 		o.KubeConfig = config
// 	}
// }

// func Master(master string) Option {
// 	return func(o *options) {
// 		o.Master = master
// 	}
// }

// type Client struct {
// 	opts   options
// 	client *kubernetes.Clientset
// }

// func NewClient(opts ...Option) *Client {
// 	op := options{}
// 	for _, o := range opts {
// 		o(&op)
// 	}
// 	return &Client{
// 		opts: op,
// 	}
// }

// func (c *Client) kubeClient() (err error) {
// 	var config *rest.Config
// 	if c.opts.KubeConfig != "" {
// 		if config, err = clientcmd.BuildConfigFromFlags(c.opts.Master, c.opts.KubeConfig); err != nil {
// 			return err
// 		}
// 	} else {
// 		if config, err = rest.InClusterConfig(); err != nil {
// 			return err
// 		}
// 	}
// 	if c.client, err = kubernetes.NewForConfig(config); err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (c *Client) Load(configMapName string) (string, error) {
// 	if c.client == nil {
// 		err := c.kubeClient()
// 		if err != nil {
// 			return "", err
// 		}
// 	}
// 	configmap, err := c.client.
// 		CoreV1().
// 		ConfigMaps(c.opts.Namespace).
// 		Get(context.Background(), configMapName, metav1.GetOptions{})
// 	if err != nil {
// 		return "", err
// 	}
// 	datamap, ok := configmap.Data["config.yaml"]
// 	if !ok {
// 		return "", fmt.Errorf("Failed to get configmap")
// 	}
// 	return datamap, nil
// }
