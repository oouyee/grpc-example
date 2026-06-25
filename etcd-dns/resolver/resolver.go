package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

const Scheme = "etcd"

type etcdBuilder struct {
	cli *clientv3.Client
	mu  sync.Mutex
	cc  resolver.ClientConn
}

type ServiceInfo struct {
	ServiceName string            `json:"service_name"`
	Address     string            `json:"address"`
	Metadata    map[string]string `json:"metadata"`
}

func NewEtcdResolver(endpoints []string) (resolver.Builder, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %v", err)
	}

	builder := &etcdBuilder{
		cli: cli,
	}

	resolver.Register(builder)
	return builder, nil
}

func (b *etcdBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	serviceName := target.Endpoint()
	prefix := fmt.Sprintf("/services/%s/", serviceName)

	r := &etcdResolver{
		serviceName: serviceName,
		prefix:      prefix,
		cli:         b.cli,
		cc:          cc,
		done:        make(chan struct{}),
	}

	r.updateAddresses()

	go r.watchChanges()

	return r, nil
}

func (b *etcdBuilder) Scheme() string {
	return Scheme
}

type etcdResolver struct {
	mu          sync.Mutex
	serviceName string
	prefix      string
	cli         *clientv3.Client
	cc          resolver.ClientConn
	done        chan struct{}
}

func (r *etcdResolver) updateAddresses() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := r.cli.Get(ctx, r.prefix, clientv3.WithPrefix())
	if err != nil {
		fmt.Printf("Failed to get services from etcd: %v\n", err)
		return
	}

	var addrs []resolver.Address
	for _, kv := range resp.Kvs {
		var svc ServiceInfo
		if err := json.Unmarshal(kv.Value, &svc); err != nil {
			fmt.Printf("Failed to unmarshal service info: %v\n", err)
			continue
		}
		if svc.Metadata == nil {
			svc.Metadata = make(map[string]string)
		}
		weight, _ := strconv.Atoi(svc.Metadata["weight"])

		addrs = append(addrs, resolver.Address{
			Addr:       svc.Address,
			Attributes: attributes.New("weight", weight),
		})
	}

	if len(addrs) == 0 {
		fmt.Printf("No services found for: %s\n", r.serviceName)
		return
	}

	state := resolver.State{
		Addresses: addrs,
	}
	r.cc.UpdateState(state)

	fmt.Printf("Updated %d addresses for service: %s\n", len(addrs), r.serviceName)
}

func (r *etcdResolver) watchChanges() {
	watchChan := r.cli.Watch(context.Background(), r.prefix, clientv3.WithPrefix())

	for {
		select {
		case <-r.done:
			return
		case watchResp, ok := <-watchChan:
			if !ok {
				return
			}

			for range watchResp.Events {
				r.updateAddresses()
				break
			}
		}
	}
}

func (r *etcdResolver) ResolveNow(opts resolver.ResolveNowOptions) {
	r.updateAddresses()
}

func (r *etcdResolver) Close() {
	close(r.done)
}

func ParseEndpoints(target string) []string {
	endpoints := strings.TrimPrefix(target, Scheme+":///")
	return strings.Split(endpoints, ",")
}
