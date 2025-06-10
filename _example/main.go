package main

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"time"

	"github.com/libdns/libdns"
	"go.megpoid.dev/libdns-linode"
)

func main() {
	token := os.Getenv("LINODE_AUTH_TOKEN")
	if token == "" {
		fmt.Printf("LINODE_AUTH_TOKEN not set\n")
		return
	}
	zone := os.Getenv("ZONE")
	if zone == "" {
		fmt.Printf("ZONE not set\n")
		return
	}
	provider := linode.Provider{APIToken: token}

	records := []libdns.Record{
		libdns.CNAME{
			Name:   "test-cname",
			TTL:    30 * time.Second,
			Target: "@",
		},
		libdns.Address{
			Name: "test-ipv4",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("127.0.0.1"),
		},
		libdns.Address{
			Name: "test-ipv6",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("::1"),
		},
		libdns.MX{
			Name:       "test-mx",
			TTL:        300 * time.Second,
			Preference: 10,
			Target:     "mail.example.com",
		},
		libdns.TXT{
			Name: "test-txt",
			TTL:  300 * time.Second,
			Text: "Sample text",
		},
		libdns.SRV{
			Name:      "test-srv",
			TTL:       300 * time.Second,
			Transport: "tcp",
			Service:   "http",
			Priority:  5,
			Weight:    10,
			Port:      8080,
			Target:    "srv.example.com.",
		},
		libdns.CAA{
			Name:  "test-caa",
			TTL:   30 * time.Second,
			Flags: 0,
			Tag:   "issue",
			Value: "letsencrypt.org",
		},
	}

	_, err := provider.AppendRecords(context.TODO(), zone, records)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	} else {
		fmt.Println("Added records")
	}
}
