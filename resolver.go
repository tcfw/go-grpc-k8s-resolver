package main

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc/resolver"
)

type k8sResolver struct {
	k8sC   k8sServiceEndpointResolver
	host   string
	port   string
	ctx    context.Context
	cancel context.CancelFunc
	cc     resolver.ClientConn
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn chan struct{}
	// wg is used to enforce Close() to return after the watcher() goroutine has finished.
	// Otherwise, data race will be possible. [Race Example] in dns_resolver_test we
	// replace the real lookup functions with mocked ones to facilitate testing.
	// If Close() doesn't wait for watcher() goroutine finishes, race detector sometimes
	// will warns lookup (READ the lookup function pointers) inside watcher() goroutine
	// has data race with replaceNetFunc (WRITE the lookup function pointers).
	wg sync.WaitGroup
}

// ResolveNow invoke an immediate resolution of the target that this k8sResolver watches.
func (k *k8sResolver) ResolveNow(resolver.ResolveNowOptions) {
	select {
	case k.rn <- struct{}{}:
	default:
	}
}

// Close closes the k8sResolver.
func (k *k8sResolver) Close() {
	k.cancel()
	k.wg.Wait()
}

func (k *k8sResolver) watcher() {
	defer k.wg.Done()

	we, err := k.k8sC.Watch(k.ctx, k.host)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-k.ctx.Done():
			return
		case <-k.rn:
		case <-we:
		}

		state, err := k.lookup()
		if err != nil {
			k.cc.ReportError(err)
		} else {
			k.cc.UpdateState(*state)
		}

		// Sleep to prevent excessive re-resolutions. Incoming resolution requests
		// will be queued in k.rn.
		t := time.NewTimer(minK8SResRate)
		select {
		case <-t.C:
		case <-k.ctx.Done():
			t.Stop()
			return
		}
	}
}

func (k *k8sResolver) lookup() (*resolver.State, error) {
	endpoints, err := k.k8sC.Resolve(k.ctx, k.host, k.port)
	if err != nil {
		return nil, err
	}

	state := &resolver.State{Addresses: []resolver.Address{}}

	for _, ep := range endpoints {
		state.Addresses = append(state.Addresses, resolver.Address{Addr: ep})
	}

	return state, nil
}
