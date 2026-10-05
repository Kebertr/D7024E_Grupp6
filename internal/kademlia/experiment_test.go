/*
package kademlia

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// experimentTransport injects loss as an immediate send error. This avoids
// waiting for the production 2-second RPC timeout for every dropped probe.
type experimentTransport struct {
	base  *mockNetwork
	loss  float64
	rng   *rand.Rand
	rngMu sync.Mutex
}

type experimentConnection struct {
	Connection
	transport *experimentTransport
}

func (transport *experimentTransport) Listen(address Address) (Connection, error) {
	connection, err := transport.base.Listen(address)
	if err != nil {
		return nil, err
	}
	return &experimentConnection{Connection: connection, transport: transport}, nil
}

func (connection *experimentConnection) Send(message Message) error {
	connection.transport.rngMu.Lock()
	lost := connection.transport.rng.Float64() < connection.transport.loss
	connection.transport.rngMu.Unlock()
	if lost {
		return errors.New("experiment packet loss")
	}
	return connection.Connection.Send(message)
}

type experimentConfiguration struct {
	nodes      int
	values     int
	latency    time.Duration
	packetLoss float64
}

type experimentResult struct {
	lookupSuccess float64
	lookupMillis  float64
	lookupProbes  float64
}

func TestExperiment(t *testing.T) {
	configurations := []experimentConfiguration{
		{nodes: 25, values: 25},
		{nodes: 50, values: 50},
		{nodes: 250, values: 150},
		{nodes: 1000, values: 500},
		{nodes: 100, values: 10, packetLoss: 0.01},
		{nodes: 100, values: 10, packetLoss: 0.05},
		{nodes: 100, values: 10, packetLoss: 0.10},
		{nodes: 100, values: 10, packetLoss: 0.20},
	}
	seeds := []int64{11, 22, 33, 44, 55}

	for _, configuration := range configurations {
		results := make([]experimentResult, 0, len(seeds))
		for _, seed := range seeds {
			results = append(results, runExperiment(configuration, seed))
		}

		successRates := make([]float64, len(results))
		lookupMillis := make([]float64, len(results))
		lookupProbes := make([]float64, len(results))
		for index, result := range results {
			successRates[index] = result.lookupSuccess
			lookupMillis[index] = result.lookupMillis
			lookupProbes[index] = result.lookupProbes
		}
		t.Logf("nodes=%d values=%d latency=%s packet_loss=%.2f seeds=%d lookup_success=%.3f +/- %.3f probes=%.3f +/- %.3f lookup_ms=%.3f +/- %.3f expected_log2N=%.3f",
			configuration.nodes, configuration.values, configuration.latency, configuration.packetLoss, len(seeds),
			mean(successRates), standardDeviation(successRates),
			mean(lookupProbes), standardDeviation(lookupProbes),
			mean(lookupMillis), standardDeviation(lookupMillis), math.Log2(float64(configuration.nodes)))
	}
}

func runExperiment(configuration experimentConfiguration, seed int64) experimentResult {
	rng := rand.New(rand.NewSource(seed))
	mock := NewMockNetworkWithSeed(seed + 1)
	mock.latency = float64(configuration.latency)
	transport := &experimentTransport{
		base: mock,
		rng:  rand.New(rand.NewSource(seed + 2)),
	}
	contacts := make([]Contact, configuration.nodes)
	nodes := make([]*Kademlia, configuration.nodes)
	ports := rng.Perm(configuration.nodes)

	for index := range contacts {
		address := fmt.Sprintf("127.0.0.1:%d", 10000+ports[index])
		contacts[index] = NewContact(NewRandomKademliaID(), address)
	}
	for index, contact := range contacts {
		network, err := initNetwork(transport, contact)
		if err != nil {
			panic(err)
		}
		nodes[index] = &Kademlia{
			Contact:      contact,
			RoutingTable: NewRoutingTable(contact),
			Network:      network,
			Data:         make(map[string][]byte),
		}
		for peerIndex, peer := range contacts {
			if index != peerIndex {
				nodes[index].RoutingTable.AddContact(peer)
			}
		}
		network.ServerListen(nodes[index])
	}
	defer func() {
		for _, node := range nodes {
			_ = node.Network.Close()
		}
	}()

	keys := make([]string, 0, configuration.values)
	for index := 0; index < configuration.values; index++ {
		value := make([]byte, 32)
		_, _ = rng.Read(value)

		if err := nodes[index%len(nodes)].Store(value); err == nil {
			keys = append(keys, NewValueID(value).String())
		}
	}
	transport.loss = configuration.packetLoss

	var successful int
	var totalLatency time.Duration
	var totalProbes int
	for index, key := range keys {
		start := time.Now()
		_, _, probes, err := nodes[(index+1)%len(nodes)].LookupDataWithProbes(key)
		totalLatency += time.Since(start)
		totalProbes += probes
		if err == nil {
			successful++
		}
	}

	if len(keys) == 0 {
		return experimentResult{}
	}
	return experimentResult{
		lookupSuccess: float64(successful) / float64(len(keys)),
		lookupMillis:  totalLatency.Seconds() * 1000 / float64(len(keys)),
		lookupProbes:  float64(totalProbes) / float64(len(keys)),
	}
}

func mean(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func standardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	average := mean(values)
	var squaredDifference float64
	for _, value := range values {
		squaredDifference += (value - average) * (value - average)
	}
	return math.Sqrt(squaredDifference / float64(len(values)-1))
}


*/