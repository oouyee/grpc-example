package main

import (
	"sync"
	"time"

	"google.golang.org/grpc/resolver"
)

const Scheme = "discovery"

type discoveryBuilder struct{}

// 每次 grpc.NewClient() 都会调用一次 Build()
func (b *discoveryBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	serviceName := target.Endpoint()

	// 1. 创建 resolver 实例
	r := &discoveryResolver{
		serviceName: serviceName,
		cc:          cc,
		done:        make(chan struct{}),
	}

	// 2. 立即获取当前服务地址列表
	r.updateAddresses()

	// 3. 启动后台监控协程（每5秒检查一次服务变化）
	go r.watchChanges()

	return r, nil
}

func (b *discoveryBuilder) Scheme() string {
	return Scheme
}

type discoveryResolver struct {
	mu          sync.Mutex
	serviceName string
	cc          resolver.ClientConn
	done        chan struct{}
	lastAddrs   []resolver.Address
}

func (r *discoveryResolver) updateAddresses() {
	services := DefaultRegistry.Discover(r.serviceName)

	if len(services) == 0 {
		return
	}

	var addrs []resolver.Address
	for _, svc := range services {
		addrs = append(addrs, resolver.Address{
			Addr: svc.Address,
		})
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastAddrs = addrs

	state := resolver.State{
		Addresses: addrs,
	}
	r.cc.UpdateState(state)
}

func (r *discoveryResolver) watchChanges() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			r.updateAddresses()
		}
	}
}

// 触发条件（由 gRPC 自动调用）
//  1. 连接失败时
//     → TCP 连接被拒绝/超时
//  2. RPC 调用失败时
//     → 所有地址都不可用
//  3. gRPC 认为需要刷新时
//     → 负载均衡器建议重新解析
//  4. 手动调用（可选）
//     → conn.Invoke(...) 前主动触发
func (r *discoveryResolver) ResolveNow(opts resolver.ResolveNowOptions) {
	// gRPC 调用此方法时，立即查询最新的服务地址
	r.updateAddresses()
}

func (r *discoveryResolver) Close() {
	close(r.done)
}

func init() {
	resolver.Register(&discoveryBuilder{})
}
