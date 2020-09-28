package k8sresolver

import (
	"context"
	"time"

	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
)

const (
	defaultPort   = "443"
	minK8SResRate = 5 * time.Second
)

var logger = grpclog.Component("k8s")

func init() {
	resolver.Register(NewBuilder())
}

// NewBuilder creates a dnsBuilder which is used to factory DNS resolvers.
func NewBuilder() resolver.Builder {
	return &k8sBuilder{}
}

type k8sBuilder struct{}

func (b *k8sBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	host, port, err := parseTarget(target.Endpoint, defaultPort)
	if err != nil {
		return nil, err
	}

	k8sc, err := newInClusterClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	k := &k8sResolver{
		k8sC:   k8sc,
		host:   host,
		port:   port,
		ctx:    ctx,
		cancel: cancel,
		cc:     cc,
		rn:     make(chan struct{}, 1),
	}

	k.wg.Add(1)
	go k.watcher()
	k.ResolveNow(resolver.ResolveNowOptions{})
	return k, nil
}

// Scheme returns the naming scheme of this resolver builder, which is "dns".
func (b *k8sBuilder) Scheme() string {
	return "k8s"
}
