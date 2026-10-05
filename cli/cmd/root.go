/*
Copyright © 2026 Viggo Härdelin, Alex Burman, Rasmus Kebert
*/
package cmd

import (
	"bufio" // buffered IO package for go
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
	"github.com/spf13/cobra"
)

var node *kademlia.Kademlia

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kademlia",
	Short: "Run a kademlia distributed hash table",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	var pingIp string
	var pingPort int

	/*
		Checks wheter another Kademlia node is reachable
		go run ./cmd ping --ip 127.0.0.1 --port 8000
	*/
	var pingCmd = &cobra.Command{
		Use:   "ping",
		Short: "Ping a node in the network",
		Run: func(cmd *cobra.Command, args []string) {
			if node == nil {
				fmt.Println("Start a node before pinging")
				return
			}

			contact := kademlia.NewContact(nil, pingIp+":"+strconv.Itoa(pingPort))
			if err := node.Ping(&contact); err != nil {
				fmt.Println("Ping failed:", err)
				return
			}
			fmt.Println("Pong")
		},
	}

	pingCmd.Flags().StringVarP(&pingIp, "ip", "i", "127.0.0.1", "IP address to ping")
	pingCmd.Flags().IntVarP(&pingPort, "port", "p", 0, "Port number to ping")
	rootCmd.AddCommand(pingCmd)

	var startIp string
	var startPort int
	var startId string

	/*
		Creates and runs a local kademlia node
		"go run ./cmd start --port 8000"
	*/
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start a node and open the interactive shell",
		Run: func(cmd *cobra.Command, args []string) {
			id := kademlia.NewKademliaID(startId)
			var err error
			node, err = kademlia.NewKademlia(id, fmt.Sprintf("%s:%d", startIp, startPort))
			if err != nil {
				fmt.Println("Unable to start node:", err)
				return
			}
			defer func() {
				if err := node.Close(); err != nil {
					fmt.Println("Unable to close node:", err)
				}
				node = nil
			}()

			bootAddress := os.Getenv("BOOT_ADDRESS")

			bootNode := kademlia.NewContact(nil, bootAddress)
			node.Ping(&bootNode)

			runShell()
		},
	}
	startCmd.Flags().StringVarP(&startIp, "ip", "i", "127.0.0.1", "IP address of this node")
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Port number of this node")
	startCmd.Flags().StringVar(&startId, "id", kademlia.NewRandomKademliaID().String(), "Kademlia ID of this node")
	rootCmd.AddCommand(startCmd)
}

func runShell() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Kademlia network started. Type 'help' for commands or 'exit' to quit.")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Println("Input error:", err)
			}
			return
		}

		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "exit", "quit", "q":
			return
		case "help":
			fmt.Println("ping <ip> <port> | put <value> | get <key> <filename>| exit | detach")
		case "ping":
			shellPing(fields[1:])
		case "put":
			contacts := node.RoutingTable.GetAllContacts()
			println(len(contacts))
			shellput(fields[0:])
		case "get":
			shellGet(fields[0:])
		case "show":
			if len(fields) != 2 {
				fmt.Println("Usage: show rt|ds")
				continue
			}

			switch fields[1] {
			case "rt":
				shellShowRoutingTable()
			case "ds":
				shellShowDataStore()
			default:
				fmt.Println("Usage: show rt|ds")
			}
		case "detach":
			fmt.Println("Use Ctrl-P, Ctrl-Q to detach.")
			return
		default:
			fmt.Println("Unknown command. Type 'help' for available commands.")
		}
	}
}

func shellPing(args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: ping <ip> <port>")
		return
	}

	port, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("Invalid port:", err)
		return
	}
	contact := kademlia.NewContact(nil, args[0]+":"+strconv.Itoa(port))

	pingTime := time.Now()
	var ping = node.Ping(&contact)
	duration := time.Since(pingTime)
	if ping != nil {
		fmt.Printf("Ping failed after %v: %v\n", duration, err)
		return
	}
	fmt.Printf("Pinging to %s in %v\n", contact.Address, duration)
}

func shellput(args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: put <value>")
		return
	}
	value := args[1]

	data, err := os.ReadFile(value)

	if err != nil {
		fmt.Println(err)
		return
	}
	err = node.Store(data)

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Stored succeded with key", kademlia.NewValueID(data))
}

func shellGet(args []string) {
	if len(args) != 2 && len(args) != 3 {
		fmt.Println("Usage: get <key> <filename>")
		return
	}
	search := args[1]

	value, sender, ok := node.LookupData(search)
	if ok != nil {
		fmt.Println("Key not found")
		return
	}

	if len(args) == 2 {
		fmt.Println("Value:", string(value))
		fmt.Println("The node that sent it:", sender.Address)
	} else {
		filename := args[2]
		err := os.WriteFile(filename, value, 0644)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Added it to file ", filename)
		fmt.Println("The node that sent it:", sender.Address)
	}
}

func shellShowRoutingTable() {
	if node == nil {
		fmt.Println("Start a node before showing the routing table")
		return
	}

	fmt.Printf("Routing table for %s (%s)\n", node.Contact.Address, node.Contact.ID)

	buckets := node.RoutingTable.YoinkContacts()
	total := 0

	for bucketIndex, contacts := range buckets {
		if len(contacts) == 0 {
			continue
		}

		fmt.Printf("\nBucket %d (%d/%d contacts)\n",
			bucketIndex, len(contacts), kademlia.BucketSize)

		for contactIndex, contact := range contacts {
			fmt.Printf("  %d. %s @ %s\n",
				contactIndex+1, contact.ID, contact.Address)
			total++
		}
	}

	if total == 0 {
		fmt.Println("\nRouting table is empty")
	} else {
		fmt.Printf("\nTotal contacts: %d\n", total)
	}
}

func shellShowDataStore() {
	if node == nil {
		fmt.Println("Start a node before showing the data store")
		return
	}

	if len(node.Data) == 0 {
		fmt.Println("Data store is empty")
		return
	}

	fmt.Println("Stored keys:")
	for key := range node.Data {
		fmt.Println(key)
	}
}
