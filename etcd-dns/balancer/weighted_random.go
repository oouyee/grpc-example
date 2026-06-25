package balancer

import (
	"math/rand"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

const WeightedRandomName = "weighted_random"

type weightedRandomPicker struct {
	subConns []balancer.SubConn
	weights  []int
	mu       sync.Mutex
}

func (p *weightedRandomPicker) Pick(pickInfo balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.subConns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	totalWeight := 0
	for _, w := range p.weights {
		totalWeight += w
	}

	if totalWeight == 0 {
		totalWeight = len(p.subConns)
		for i := range p.weights {
			p.weights[i] = 1
		}
	}

	r := rand.Intn(totalWeight)
	sum := 0
	for i, weight := range p.weights {
		sum += weight
		if r < sum {
			return balancer.PickResult{
				SubConn: p.subConns[i],
			}, nil
		}
	}

	return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
}

type weightedRandomPickerBuilder struct{}

func (p *weightedRandomPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	var subConns []balancer.SubConn
	var weights []int

	for sc, scInfo := range info.ReadySCs {
		subConns = append(subConns, sc)
		weights = append(weights, extractWeight(scInfo.Address))
	}

	return &weightedRandomPicker{
		subConns: subConns,
		weights:  weights,
	}
}

func extractWeight(addr resolver.Address) int {
	if addr.Attributes == nil {
		return 0
	}

	if w, ok := addr.Attributes.Value("weight").(int); ok {
		if w > 0 {
			return w
		}
	}

	return 1
}

type weightedRandomBuilder struct{}

func (b *weightedRandomBuilder) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	return base.NewBalancerBuilder(
		WeightedRandomName,
		&weightedRandomPickerBuilder{},
		base.Config{},
	).Build(cc, opts)
}

func (b *weightedRandomBuilder) Name() string {
	return WeightedRandomName
}

func init() {
	balancer.Register(&weightedRandomBuilder{})
}
