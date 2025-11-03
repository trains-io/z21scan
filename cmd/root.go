package cmd

// TODO:
// * add Makefile for local builds, format, vet, ...
// * add github actions to publish binary
// * add README.md
// * tag version

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

const (
	MAX_CONCURRENCY int = 200
)

var (
	port    int
	output  string
	quiet   bool
	verbose bool
)

var validOutputFormats = []string{"short", "normal", "verbose", "json"}

type ScanResult struct {
	IP        net.IP `json:"ip"`
	Port      int    `json:"port"`
	Reachable bool   `json:"reachable"`
	Serial    string `json:"serial"`
}

func isValidOutput(val string) bool {
	for _, v := range validOutputFormats {
		if v == val {
			return true
		}
	}
	return false
}

func netFromIface(name string) (*net.IPNet, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			return ipnet, nil
		}
	}
	return nil, fmt.Errorf("no IPv4 network")
}

func ipsInNet(n *net.IPNet) []net.IP {
	var ips []net.IP
	for ip := n.IP.Mask(n.Mask); n.Contains(ip); incIP(ip) {
		ips = append(ips, append(net.IP(nil), ip...))
	}

	if len(ips) > 2 {
		return ips[1 : len(ips)-1]
	}
	return ips
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

func probeUDP(ip net.IP, port int, t time.Duration) (ScanResult, error) {
	res := ScanResult{
		IP:   ip,
		Port: port,
	}

	url := net.JoinHostPort(ip.String(), fmt.Sprint(port))
	conn, err := z21.Connect(url)
	if err != nil {
		return res, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()

	r, err := conn.SendRcv(ctx, &z21.SerialNumber{})
	if err != nil {
		return res, err
	}

	m, ok := r.(*z21.SerialNumber)
	if !ok {
		return res, fmt.Errorf("failed to read z21 reply")
	}

	res.Reachable = true
	res.Serial = fmt.Sprintf("%d", m.SerialNumber)

	return res, nil
}

var rootCmd = &cobra.Command{
	Use:   "z21scan [IFACE|NETWORK]",
	Short: "Scan local network for Z21 devices.",
	Long: `z21scan scans a local network for reachable Z21 devices.
You can specify either a network interface (e.g. "eth0") or a 
network address in CIDR notation (e.g. "192.168.2.0/24").`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  false,
	SilenceErrors: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if quiet {
			output = "short"
		} else if verbose {
			output = "verbose"
		}
		if !isValidOutput(output) {
			return fmt.Errorf("invalid output format: %q (valid: %v)", output, validOutputFormats)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		var netaddr *net.IPNet
		var err error

		if strings.Contains(target, "/") {
			_, netaddr, err = net.ParseCIDR(target)
			if err != nil {
				return fmt.Errorf("invalid network address: %v", err)
			}
		} else {
			netaddr, err = netFromIface(target)
			if err != nil {
				return fmt.Errorf("failed to get network address from interface %q: %v", target, err)
			}
		}

		ips := ipsInNet(netaddr)
		resultsCh := make(chan ScanResult, len(ips))
		sem := make(chan struct{}, MAX_CONCURRENCY)
		var wg sync.WaitGroup

		if output == "normal" || output == "verbose" {
			fmt.Printf("Scanning network %q (port: %d) ...\n", netaddr, port)
		}
		for _, ip := range ips {
			wg.Add(1)
			sem <- struct{}{}
			go func(target net.IP) {
				defer wg.Done()
				defer func() {
					<-sem
				}()
				result, _ := probeUDP(ip, port, 2*time.Second)
				if output == "verbose" {
					fmt.Printf("Probing %-14s -> z21 device: %t\n", ip, result.Reachable)
				}
				resultsCh <- result
			}(ip)
		}

		wg.Wait()
		close(resultsCh)

		var results []ScanResult
		for r := range resultsCh {
			if r.Reachable {
				results = append(results, r)
			}
		}

		switch output {
		case "short":
			for _, r := range results {
				fmt.Println(r.IP)
			}

		case "normal", "verbose":
			fmt.Printf("Found %d Z21 device(s)\n", len(results))
			for _, r := range results {
				fmt.Printf("  %-15s port=%d serial=%s\n", r.IP, r.Port, r.Serial)
			}

		case "json":
			if results == nil {
				results = []ScanResult{}
			}
			b, err := json.Marshal(results)
			if err != nil {
				return fmt.Errorf("failed to marshall results to JSON: %v", err)
			}
			fmt.Println(string(b))
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 21105, "UDP port to probe")
	rootCmd.Flags().StringVarP(&output, "output", "o", "normal", "Output format: short|normal|verbose|json")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Short output (same as -o short)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (same as -o verbose)")
}
