package kademlia

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

type experiment struct {
	loss        float64
	latency     int
	seed        int64
	networkSize int
	mu          sync.Mutex
}

type configuration struct {
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

		nodes[i] = &Kademlia{
			Contact:      contact,
			RoutingTable: NewRoutingTable(contact),
			Network:      nodeNetwork,
			Data:         make(map[string][]byte),
		}

		nodeNetwork.ServerListen(nodes[i])
	}

	// Give every node a random initial routing table.
	for i := range nodes {
		order := rng.Perm(len(nodes))
		added := 0

		for _, candidate := range order {
			if candidate == i {
				continue
			}

			nodes[i].RoutingTable.AddContact(nodes[candidate].Contact)
			added++

			// Keep the initial topology sparse.
			if added == BucketSize {
				break
			}
		}
	}

	network.latency = float64(config.latency)
	network.packet_loss = config.packetLoss

	return network, nodes, nil
}

func TestExperiment(t *testing.T) {
	seeds := []int64{1, 2, 3, 4, 5}
	configs := []configuration{
		// None
		{nodes: 100, values: 50, latency: 0, packetLoss: 0.0},

		// PL
		{nodes: 100, values: 50, latency: 0, packetLoss: 0.01},
		{nodes: 100, values: 50, latency: 0, packetLoss: 0.05},
		{nodes: 100, values: 50, latency: 0, packetLoss: 0.1},

		// Latency
		{nodes: 100, values: 50, latency: 0 * time.Millisecond, packetLoss: 0},
		{nodes: 100, values: 50, latency: 100 * time.Millisecond, packetLoss: 0},
		{nodes: 100, values: 50, latency: 250 * time.Millisecond, packetLoss: 0},

		// Both
		{nodes: 100, values: 50, latency: 100 * time.Millisecond, packetLoss: 0.01},
		{nodes: 100, values: 50, latency: 250 * time.Millisecond, packetLoss: 0.05},
		{nodes: 100, values: 50, latency: 500 * time.Millisecond, packetLoss: 0.1},
	}

	for _, config := range configs {
		for _, seed := range seeds {
			network, nodes, err := buildNet(config, seed)
			if err != nil {
				t.Fatal(err)
			}

			rng := rand.New(rand.NewSource(seed + 1000))
			successes := 0
			totalTime := time.Duration(0)

			for lookup := 0; lookup < config.values; lookup++ {
				source := nodes[rng.Intn(len(nodes))]
				target := nodes[rng.Intn(len(nodes))]

				start := time.Now()
				contacts, err := source.LookupContact(&target.Contact)
				totalTime += time.Since(start)

				if err == nil && len(contacts) > 0 {
					successes++
				}
			}

			successRate := float64(successes) / float64(config.values)
			averageTime := totalTime / time.Duration(config.values)

			t.Logf(
				"seed=%d nodes=%d latency=%s packet_loss=%.2f success_rate=%.3f average_lookup_time=%s",
				seed,
				len(nodes),
				config.latency,
				config.packetLoss,
				successRate,
				averageTime,
			)

			// Close this network before creating the next one.
			for _, node := range nodes {
				_ = node.Network.Close()
			}

			_ = network
		}
	}
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

Examples of additional optional experiments:

    Time between request and response as a function of packet loss and/or latency.
    Number of lookup probes and time required for lookups as a function of alpha (particularly important to discuss what we should expect).
    Lookup reliability as a function of churn rate. (Requires that you can control the churn rate, i.e. when nodes join/leave the network.)
    Lookup reliability as a function of the replication factor k.

Note that some of these experiments are affected by your RPC timeout/retry policy (i.e. how long you wait for a response and whether/how often you retry), which is therefore important to document clearly.
*/
