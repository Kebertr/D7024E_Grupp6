package kademlia

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type configuration struct {
	nodes            int
	values           int
	latency          time.Duration
	packetLoss       float64
	fullRoutingTable bool
}

type experimentConfigurations struct {
	scalability []configuration
	reliability []configuration
	loss        []configuration
}

func getConf() experimentConfigurations {
	return experimentConfigurations{
		scalability: []configuration{
			{nodes: 10, values: 10, packetLoss: 0, latency: 0, fullRoutingTable: true},
			{nodes: 50, values: 10, packetLoss: 0, latency: 0, fullRoutingTable: true},
			{nodes: 100, values: 10, packetLoss: 0, latency: 0, fullRoutingTable: true},
			{nodes: 250, values: 10, packetLoss: 0, latency: 0, fullRoutingTable: true},
			{nodes: 500, values: 10, packetLoss: 0, latency: 0, fullRoutingTable: true},
			{nodes: 1000, values: 10, packetLoss: 0, latency: 0, fullRoutingTable: true},
		},
		reliability: []configuration{
			{nodes: 100, values: 10, packetLoss: 0, latency: 50 * time.Millisecond},
			{nodes: 100, values: 10, packetLoss: 0, latency: 150 * time.Millisecond},
			{nodes: 100, values: 10, packetLoss: 0, latency: 250 * time.Millisecond},
			{nodes: 100, values: 10, packetLoss: 0, latency: 500 * time.Millisecond},
		},
		loss: []configuration{
			{nodes: 100, values: 10, packetLoss: 0.1, latency: 0},
			{nodes: 100, values: 10, packetLoss: 0.2, latency: 0},
			{nodes: 100, values: 10, packetLoss: 0.35, latency: 0},
			{nodes: 100, values: 10, packetLoss: 0.5, latency: 0},
		},
	}
}

// Build net and pass it to every node
func buildNet(config configuration, seed int64) (*mockNetwork, []*Kademlia, error) {
	// rng seed, same seed produce same result (like minecraft seed)
	rng := rand.New(rand.NewSource(seed))
	network := NewMockNetwork()

	// Use unique, repeatable pseudo-random ports.
	ports := rng.Perm(55536)

	nodes := make([]*Kademlia, config.nodes)

	for i := range nodes {
		var id KademliaID
		if _, err := rng.Read(id[:]); err != nil {
			return nil, nil, err
		}

		contact := NewContact(
			&id,
			fmt.Sprintf("127.0.0.1:%d", 10000+ports[i]),
		)

		nodeNetwork, err := initNetwork(network, contact)
		if err != nil {
			return nil, nil, err
		}
		nodeNetwork.rpcTimeout = 1500 * time.Millisecond
		if config.packetLoss > 0 {
			nodeNetwork.rpcTimeout = 100 * time.Millisecond
		}

		nodes[i] = &Kademlia{
			Contact:      contact,
			RoutingTable: NewRoutingTable(contact),
			Network:      nodeNetwork,
			Data:         make(map[string][]byte),
		}

		nodeNetwork.ServerListen(nodes[i])
	}

	if config.fullRoutingTable {
		for i := range nodes {
			for j := range nodes {
				if i == j {
					continue
				}

				bucketIndex := nodes[i].RoutingTable.getBucketIndex(nodes[j].Contact.ID)
				nodes[i].RoutingTable.buckets[bucketIndex].List.PushFront(nodes[j].Contact)
			}
		}
	} else {
		const extraNeighbors = 2

		for i := range nodes {
			if len(nodes) < 2 {
				continue
			}

			nodes[i].RoutingTable.AddContact(
				nodes[(i+len(nodes)-1)%len(nodes)].Contact,
			)
			nodes[i].RoutingTable.AddContact(
				nodes[(i+1)%len(nodes)].Contact,
			)

			for added := 0; added < extraNeighbors; {
				randomIndex := rng.Intn(len(nodes))
				if randomIndex == i {
					continue
				}

				nodes[i].RoutingTable.AddContact(nodes[randomIndex].Contact)
				added++
			}
		}
	}

	network.latency = float64(config.latency)
	network.packet_loss = config.packetLoss

	return network, nodes, nil
}
func runConfs(t *testing.T, config configuration, seeds []int64) {
	t.Helper()

	var totalSuccesses, totalFailures, totalProbes int64
	var totalTime time.Duration

	for _, seed := range seeds {
		_, nodes, err := buildNet(config, seed)
		if err != nil {
			t.Fatal(err)
		}

		rng := rand.New(rand.NewSource(seed + 1000))
		seedTime := time.Duration(0)

		for lookup := 0; lookup < config.values; lookup++ {
			source := nodes[rng.Intn(len(nodes))]
			target := nodes[rng.Intn(len(nodes))]

			start := time.Now()
			_, _ = source.LookupContact(&target.Contact)
			seedTime += time.Since(start)
		}

		for _, node := range nodes {
			successes, failures, probes := node.GetStats()
			totalSuccesses += successes
			totalFailures += failures
			totalProbes += probes
			_ = node.Network.Close()
		}

		totalTime += seedTime
	}

	totalLookups := totalSuccesses + totalFailures
	successRate := float64(0)
	averageProbes := float64(0)
	averageLookupTime := time.Duration(0)

	if totalLookups > 0 {
		successRate = float64(totalSuccesses) / float64(totalLookups)
		averageProbes = float64(totalProbes) / float64(totalLookups)
		averageLookupTime = totalTime / time.Duration(totalLookups)
	}

	t.Logf(
		"nodes=%d latency=%s packet_loss=%.3f seeds=%d lookups=%d success_rate=%.3f average_probes=%.2f average_lookup_time=%s",
		config.nodes,
		config.latency,
		config.packetLoss,
		len(seeds),
		totalLookups,
		successRate,
		averageProbes,
		averageLookupTime,
	)
}

func TestExperiment(t *testing.T) {
	seeds := []int64{1, 2, 3}
	configs := getConf()

	t.Run("lookup scalability", func(t *testing.T) {
		for _, config := range configs.scalability {
			runConfs(t, config, seeds)
		}
	})

	t.Run("lookup reliability", func(t *testing.T) {
		for _, config := range configs.reliability {
			runConfs(t, config, seeds)
		}
	})

	t.Run("lookup loss", func(t *testing.T) {
		for _, config := range configs.loss {
			runConfs(t, config, seeds)
		}
	})
}

/*
Experimental Evaluation

Your report will present results for at least two experiments. You collect the data by logging events (as discussed above under the instrumentation requirement) and analyzing the log with a simple, external script. You may use any language for the script (which is exempt from the code coverage requirement). Make the log entries simple and structured, i.e. machine readable without requiring unnecessarily complex parsing.

Basic methodology rules:

    Randomly generate a network topology (i.e. different IP:port combinations so we get different node IDs)
    Randomly generate key-value pairs
    Make runs repeatable by explicitly setting the RNG seeds.
    Repeat the experiment for each configuration (different parameters) with multiple seeds.
    Report variance along with an average.
    Explain the experimental setup. In particular, how were things measured? Why should we believe those measurements are meaningful?
    Explain what results we should expect and why. Compare the observed result to the expectation and discuss (i.e. try to explain) any deviations.

Mandatory experiments:

    Lookup scalability as a function of the network size N:
        Plot the number of probes needed during lookups as a function of N.
        Compare to the expected number of probes/"hops".
    Lookup reliability (success rate) as a function of packet loss probability.

Note that some of these experiments are affected by your RPC timeout/retry policy (i.e. how long you wait for a response and whether/how often you retry), which is therefore important to document clearly.
*/
