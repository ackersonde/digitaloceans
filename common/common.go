package common

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

// TokenSource is now commented
type TokenSource struct {
	AccessToken string
}

// Token is now commented
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

var doPersonalAccessToken = os.Getenv("CTX_DIGITALOCEAN_DROPLET_PROXY_TOKEN")
var firewallID = os.Getenv("CTX_DIGITALOCEAN_FIREWALL")

// FloatingIPAddress is the static IP for ackerson.de
var FloatingIPAddress = os.Getenv("doFloatingIP")

// PrepareDigitalOceanLogin does what it says on the box
func PrepareDigitalOceanLogin() *godo.Client {
	tokenSource := &TokenSource{
		AccessToken: doPersonalAccessToken,
	}

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	return godo.NewClient(oauthClient)
}

func prepareSSHipAddresses() []string {
	ipAddys := []string{}
	ipAddrs, _ := net.LookupIP(os.Getenv("homeDomain"))
	for _, ipAddr := range ipAddrs {
		ipAddys = append(ipAddys, ipAddr.String())
	}

	return ipAddys
}

// ToggleSSHipAddress adds/removes an IP address on the FW rule
func ToggleSSHipAddress(add bool, ipAddress string, client *godo.Client) {
	ctx := context.Background()

	ruleRequest := &godo.FirewallRulesRequest{
		InboundRules: []godo.InboundRule{
			{
				Protocol:  "tcp",
				PortRange: "22",
				Sources: &godo.Sources{
					Addresses: []string{ipAddress},
				},
			},
		},
	}

	if add {
		_, err := client.Firewalls.AddRules(ctx, firewallID, ruleRequest)
		if err != nil {
			log.Println(err)
		}
	} else {
		_, err := client.Firewalls.RemoveRules(ctx, firewallID, ruleRequest)
		if err != nil {
			log.Println(err)
		}
	}
}

func GetSSHFirewallRules() []string {
	var sshSources []string
	client := PrepareDigitalOceanLogin()
	firewall, _, _ := client.Firewalls.Get(context.Background(), firewallID)
	for _, rule := range firewall.InboundRules {
		if rule.PortRange == "22" {
			sshSources = append(sshSources, rule.Sources.Addresses...)
		}
	}

	return sshSources
}

func FindExistingDeployDroplet(tag string) godo.Droplet {
	client := PrepareDigitalOceanLogin()
	ctx := context.Background()
	var droplet godo.Droplet

	droplet.ID = 1 // set default, nonsensical value
	if tag == "traefik" {
		droplets, _, _ := client.Droplets.ListByTag(ctx, tag, &godo.ListOptions{})
		if len(droplets) > 0 {
			droplet = droplets[0]
		}
	}

	return droplet
}

func UpdateDNSentry(ipAddr string, hostname string, domainID int) {
	client := PrepareDigitalOceanLogin()
	record, _, err := client.Domains.Record(context.Background(), hostname, domainID)
	if err != nil {
		log.Printf("unable to updateDNS for %s: %s", hostname, err.Error())
	}
	//fmt.Printf("current DNS %s: %s => %s\n", record.Name, record.Type, record.Data)

	editRequest := &godo.DomainRecordEditRequest{
		Type: record.Type,
		Name: record.Name,
		Data: strings.ToLower(ipAddr),
	}
	_, _, err = client.Domains.EditRecord(context.Background(), hostname, domainID, editRequest)
	for err != nil {
		fmt.Printf("FAIL domain update DNS: %s\n", err)
		time.Sleep(5 * time.Second)
		_, _, err = client.Domains.EditRecord(context.Background(), hostname, domainID, editRequest)
	}
}

func UpdateFirewall() {
	ipAddys := prepareSSHipAddresses()

	client := PrepareDigitalOceanLogin()
	ctx := context.Background()

	floatingIP, _, err := client.FloatingIPs.Get(ctx, os.Getenv("doFloatingIP"))
	if err != nil {
		log.Println(err)
	}
	for floatingIP.Droplet == nil {
		if err != nil {
			log.Println(err)
		}

		log.Println("floatIP not yet assigned...")
		time.Sleep(5 * time.Second)
		floatingIP, _, err = client.FloatingIPs.Get(ctx, os.Getenv("doFloatingIP"))
	}
	log.Println("update firewall for droplet: " + strconv.Itoa(floatingIP.Droplet.ID))

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
		DropletIDs: []int{floatingIP.Droplet.ID},
	}

	firewallResp, _, err := client.Firewalls.Update(ctx, firewallID, updateRequest)
	if err == nil {
		log.Println(firewallResp)
	} else {
		log.Println(err)
	}
}

func DropletList(client *godo.Client) ([]godo.Droplet, error) {
	list := []godo.Droplet{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(context.Background(), opt)
		if err != nil {
			return nil, err
		}

		// append the current page's droplets to our list
		list = append(list, droplets...)

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

// DeleteDODroplet more here https://developers.digitalocean.com/documentation/v2/#delete-a-droplet
func DeleteDODroplet(ID int) string {
	var result string

	client := PrepareDigitalOceanLogin()

	_, err := client.Droplets.Delete(context.Background(), ID)
	if err == nil {
		result = "Successfully deleted Droplet `" + strconv.Itoa(ID) + "`"
	} else {
		result = err.Error()
	}

	return result
}

// DeleteSSHKey more here https://developers.digitalocean.com/documentation/v2/#delete-a-key
func DeleteSSHKey(ID int, client *godo.Client) string {
	var result string

	_, err := client.Keys.DeleteByID(context.Background(), ID)
	if err == nil {
		result = "Successfully deleted SSH Key `" + strconv.Itoa(ID) + "`"
	} else {
		result = err.Error()
	}

	return result
}
