package register

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdRegister struct {
	cli     *clientv3.Client
	leaseID clientv3.LeaseID
	ctx     context.Context
	cancel  context.CancelFunc
}

type ServiceInfo struct {
	ServiceName string            `json:"service_name"`
	Address     string            `json:"address"`
	Metadata    map[string]string `json:"metadata"`
}

func NewEtcdRegister(endpoints []string) (*EtcdRegister, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EtcdRegister{
		cli:    cli,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (r *EtcdRegister) Register(service *ServiceInfo, ttl int64) error {
	key := fmt.Sprintf("/services/%s/%s", service.ServiceName, service.Address)
	value, _ := json.Marshal(service)

	resp, err := r.cli.Grant(r.ctx, ttl)
	if err != nil {
		return fmt.Errorf("failed to grant lease: %v", err)
	}

	r.leaseID = resp.ID

	_, err = r.cli.Put(r.ctx, key, string(value), clientv3.WithLease(resp.ID))
	if err != nil {
		return fmt.Errorf("failed to put key: %v", err)
	}

	ch, err := r.cli.KeepAlive(r.ctx, resp.ID)
	if err != nil {
		return fmt.Errorf("failed to keep alive: %v", err)
	}

	go func() {
		for range ch {
			select {
			case <-r.ctx.Done():
				return
			default:
			}
		}
	}()

	fmt.Printf("Registered service: %s at %s (TTL: %ds)\n", service.ServiceName, service.Address, ttl)
	return nil
}

func (r *EtcdRegister) Unregister(serviceName, address string) error {
	key := fmt.Sprintf("/services/%s/%s", serviceName, address)
	_, err := r.cli.Delete(r.ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete key: %v", err)
	}

	fmt.Printf("Unregistered service: %s at %s\n", serviceName, address)
	return nil
}

func (r *EtcdRegister) Close() {
	r.cancel()
	if r.cli != nil {
		r.cli.Close()
	}
}
