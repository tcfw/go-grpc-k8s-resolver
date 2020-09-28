package main

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type serviceEndpointResolver interface {
	Resolve(ctx context.Context, host string, port string) ([]string, error)
	Watch(ctx context.Context, host string) (<-chan watch.Event, error)
}

type serviceClient struct {
	k8s *kubernetes.Clientset
}

func newInClusterClient() (*serviceClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &serviceClient{k8s: clientset}, nil
}

func (s *serviceClient) Resolve(ctx context.Context, host string, port string) ([]string, error) {
	eps := []string{}

	ep, err := s.k8s.CoreV1().Endpoints("").Get(ctx, host, metav1.GetOptions{})
	if err != nil {
		return eps, err
	}

	for _, v := range ep.Subsets {
		for _, addr := range v.Addresses {
			eps = append(eps, addr.Hostname)
		}
	}

	return eps, nil
}

func (s *serviceClient) Watch(ctx context.Context, host string) (<-chan watch.Event, error) {
	watcher, err := s.k8s.CoreV1().Endpoints("").Watch(ctx, metav1.ListOptions{Watch: true, LabelSelector: host})
	if err != nil {
		return nil, err
	}

	return watcher.ResultChan(), nil
}
