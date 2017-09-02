package common

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

// UpdateFirewall to maintain connectivity while Telekom rotates IPs
func UpdateFirewall(client *godo.Client) {
	firewallID := os.Getenv("doFirewallID")
	floatingIPAddress := os.Getenv("doFloatingIP")

	ipAddys := []string{os.Getenv("officeIP")}
	ipAddrs, _ := net.LookupIP(os.Getenv("homeDomain"))
	for _, ipAddr := range ipAddrs {
		ipAddys = append(ipAddys, ipAddr.String())
	}

	ctx := context.TODO()

	dropletID := 0
	floatingIP, _, err := client.FloatingIPs.Get(ctx, floatingIPAddress)
	if err == nil && floatingIP.Droplet != nil {
		dropletID = floatingIP.Droplet.ID
	}
	updateRequest := &godo.FirewallRequest{
		Name: "SSH-HTTP-regulation",
		InboundRules: []godo.InboundRule{
			{
				Protocol:  "tcp",
				PortRange: "80",
				Sources: &godo.Sources{
					Addresses: []string{"0.0.0.0/0", "::/0"},
				},
			},
			{
				Protocol:  "tcp",
				PortRange: "443",
				Sources: &godo.Sources{
					Addresses: []string{"0.0.0.0/0", "::/0"},
				},
			},
			{
				Protocol:  "tcp",
				PortRange: "22",
				Sources: &godo.Sources{
					Addresses: ipAddys,
				},
			},
		},
		OutboundRules: []godo.OutboundRule{
			{
				Protocol:  "tcp",
				PortRange: "1-65535",
				Destinations: &godo.Destinations{
					Addresses: []string{"0.0.0.0/0", "::/0"},
				},
			},
			{
				Protocol: "icmp",
				Destinations: &godo.Destinations{
					Addresses: []string{"0.0.0.0/0", "::/0"},
				},
			},
			{
				Protocol:  "udp",
				PortRange: "1-65535",
				Destinations: &godo.Destinations{
					Addresses: []string{"0.0.0.0/0", "::/0"},
				},
			},
		},
		DropletIDs: []int{dropletID},
	}

	firewallResp, _, err := client.Firewalls.Update(ctx, firewallID, updateRequest)
	if err == nil {
		log.Println(firewallResp)
	} else {
		log.Println(err)
	}
}

// DropletList does what it says on the box
func DropletList(client *godo.Client) ([]godo.Droplet, error) {
	list := []godo.Droplet{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(oauth2.NoContext, opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		for _, d := range droplets {
			list = append(list, d)
		}

		// if we are at the last page, break out the for loop
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}

		// set the page we want for the next request
		opt.Page = page + 1
	}

	return list, nil
}
