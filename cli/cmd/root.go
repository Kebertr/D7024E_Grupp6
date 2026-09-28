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
	Use:   "D7024E_Grupp6 kademlia",
	Short: "d7024e kademlia distributed network lab",
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
	var bootstrapAddrs []string

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

			if len(bootstrapAddrs) > 0 {
				fmt.Println("Bootstrap addresses are not supported by the current Kademlia API")
			}
			runShell()
		},
	}
	startCmd.Flags().StringVarP(&startIp, "ip", "i", "127.0.0.1", "IP address of this node")
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Port number of this node")
	startCmd.Flags().StringVar(&startId, "id", kademlia.NewRandomKademliaID().String(), "Kademlia ID of this node")
	startCmd.Flags().StringSliceVar(&bootstrapAddrs, "bootstrap", nil, "Bootstrap addresses")
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
			fmt.Println("ping <ip> <port> | put <key> <value> | get <key> | exit")
		case "ping":
			shellPing(fields[1:])
		case "put":
			if len(fields) < 3 {
				fmt.Println("Usage: put <key> <value>")
				continue
			}
			node.Data[fields[1]] = []byte(strings.Join(fields[2:], " "))
			fmt.Println("Stored", fields[1])
		case "get":
			if len(fields) != 2 {
				fmt.Println("Usage: get <key>")
				continue
			}
			value, ok := node.Data[fields[1]]
			if !ok {
				fmt.Println("Key not found")
				continue
			}
			fmt.Println("Value:", string(value))
		case "show rt":
			//TODO
		case "show dt":
			//TODO
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
	fmt.Printf("Pinging from %s in %v\n", contact.Address, duration)
}
