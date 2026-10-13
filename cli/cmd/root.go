/*
Copyright © 2026 Viggo Härdelin, Alex Burman, Rasmus Kebert
*/
package cmd

import (
	"bufio" // buffered IO package for go
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia"
	"github.com/RasmusKebert/D7024E_Grupp6/internal/package_registry"
	"github.com/spf13/cobra"
)

var node *kademlia.Kademlia
var reg *package_registry.Registry
var dns *package_registry.MockDNS

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

	/*
		Creates and runs a local kademlia node
		"go run ./cmd start --port 8000"
	*/
	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start a node and open the interactive shell",
		Run: func(cmd *cobra.Command, args []string) {
			// Parse the input as an IP address; reject hostnames and invalid IP text.
			ip := net.ParseIP(startIp)
			if ip == nil {
				fmt.Println("Invalid IP address:", startIp)
				return
			}
			if startPort < 1 || startPort > 65535 {
				fmt.Println("Port must be between 1 and 65535")
				return
			}

			id := kademlia.IPPortDerivedNodeID(ip.String(), startPort)
			var err error
			address := net.JoinHostPort(ip.String(), strconv.Itoa(startPort))
			node, err = kademlia.NewKademlia(id, address)
			if err != nil {
				fmt.Println("Unable to start node:", err)
				return
			}
			dns = package_registry.NewMockDNS()

			domains := []string{
				"example1.com",
				"example2.com",
				"example3.com",
			}

			publickey, errPublic := loadPublicKey("keys/example1.com-public.pem")

			if errPublic != nil {
				fmt.Println(errPublic)
			}

			dns.AddDomainCli(domains[0], *publickey)

			publickey, errPublic = loadPublicKey("keys/example2.com-public.pem")

			if errPublic != nil {
				fmt.Println(errPublic)
			}

			dns.AddDomainCli(domains[1], *publickey)

			publickey, errPublic = loadPublicKey("keys/example3.com-public.pem")

			if errPublic != nil {
				fmt.Println(errPublic)
			}

			dns.AddDomainCli(domains[2], *publickey)

			if err != nil {
				fmt.Println("Unable to create mock DNS:", err)
				return
			}
			reg = &package_registry.Registry{
				Kademlia: node,
				DNS:      dns,
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
	// Use a fixed default because port zero is assigned only after binding.
	startCmd.Flags().IntVarP(&startPort, "port", "p", 8000, "Port number of this node")
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
			fmt.Println("ping <ip> <port> | put <value> | get <key> <filename>| exit | detach || show rt|ds|dns DOMAIN|version DOMAIN:PACKAGE | publish [--force] [--prev=PREVIOUS-VERSION] DOMAIN:PACKAGE:VERSION FILENAME | install DOMAIN:PACKAGE:VERSION")
		case "ping":
			shellPing(fields[1:])
		case "put":
			contacts := node.RoutingTable.YoinkContacts()
			println(len(contacts))
			shellput(fields[0:])
		case "get":
			shellGet(fields[0:])
		case "show":

			switch fields[1] {
			case "rt":
				shellShowRoutingTable()
			case "ds":
				shellShowDataStore()
			case "dns":
				shellShowDNS(fields[2])
			case "version":
				shellShowVersionPackage(fields[2])
			default:
				fmt.Println("Usage: show rt|ds|dns DOMAIN|version DOMAIN:PACKAGE")
			}
		case "publish":
			shellPublish(fields[1:])

		case "install":
			shellInstall(fields[1:])

		case "cat":
			shellCat(fields[1:])
		case "detach":
			fmt.Println("Use Ctrl-P, Ctrl-Q to detach.")

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

func shellPublish(args []string) {
	force := false
	previous := ""
	filename := ""
	domain := ""
	packageName := ""
	version := ""

	switch len(args) {
	case 2:
		parts := strings.Split(args[0], ":")
		domain = parts[0]
		packageName = parts[1]
		version = parts[2]
		filename = args[1]
	default:
		var domainStruct string
		for _, arg := range args {
			if arg == "--force" {
				force = true
				continue
			}

			if strings.HasPrefix(arg, "--prev=") {
				previous = strings.TrimPrefix(arg, "--prev=")
				continue
			}

			if strings.Contains(arg, ":") {
				domainStruct = arg
				continue
			}

			filename = arg
		}
		parts := strings.Split(domainStruct, ":")
		domain = parts[0]
		packageName = parts[1]
		version = parts[2]
	}

	file, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Could not read file:", err)
		return
	}

	keyPath := "keys/" + domain + "-private.pem"
	privateKey, errPrivate := loadPrivateKey(keyPath)
	if errPrivate != nil {
		fmt.Println(errPrivate)
		return
	}

	errPublish := reg.Publish(domain, packageName, version, file, force, previous, *privateKey)

	if errPublish != nil {
		fmt.Println(errPublish)
	}

}

func shellInstall(args []string) {
	parts := strings.Split(args[0], ":")
	domain := parts[0]
	pkg := parts[1]
	version := parts[2]

	data, err := reg.Install(domain, pkg, version)
	if err != nil {
		fmt.Println("Install failed:", err)
		return
	}

	filename := pkg + ".pkg"
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Println("Write failed:", err)
		return
	}
	fmt.Println("Installed to", filename)
}

func shellShowDNS(domain string) {
	if node == nil {
		fmt.Println("Start a node before showing the DNS entry")
		return
	}

	key, err := reg.ShowDNS(domain)
	if err != nil {
		fmt.Println("DNS lookup failed:", err)
		return
	}

	fmt.Printf("DNS for %s: %v\n", domain, key)
}
func shellShowVersionPackage(domainPackage string) {
	if node == nil {
		fmt.Println("Start a node before showing the version of a package")
		return
	}

	parts := strings.Split(domainPackage, ":")
	if len(parts) != 2 {
		fmt.Println("Usage: show version DOMAIN:PACKAGE")
		return
	}

	domain := parts[0]
	pkg := parts[1]

	version, err := reg.ShowVersionPackage(domain, pkg)
	if err != nil {
		fmt.Println("Version lookup failed:", err)
		return
	}

	fmt.Printf("Version for %s:%s: %s\n", domain, pkg, version)

}

func shellCat(args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: cat FILENAME")
		return
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Println("Read failed:", err)
		return
	}

	fmt.Println(string(data))
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decoded, _ := pem.Decode(data)
	if decoded == nil {
		return nil, err
	}

	key, err := x509.ParsePKCS8PrivateKey(decoded.Bytes)
	if err != nil {
		return nil, err
	}

	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, err
	}

	return privateKey, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decoded, _ := pem.Decode(data)
	if decoded == nil {
		return nil, err
	}

	key, err := x509.ParsePKIXPublicKey(decoded.Bytes)
	if err != nil {
		return nil, err
	}

	return key.(*rsa.PublicKey), nil
}
