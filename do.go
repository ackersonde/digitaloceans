package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danackerson/digitalocean/common"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var doDropletInfoSite = "https://cloud.digitalocean.com/droplets/"
var digitalOceanSSHLoginPubKey = os.Getenv("CTX_DIGITALOCEAN_SSH_PUBKEY")
var githubBuild = os.Getenv("GITHUB_RUN_ID")

func main() {
	client := common.PrepareDigitalOceanLogin()

	fnPtr := flag.String("fn", "createNewServer|deleteServer|toggleSSHFirewall", "which function to run")

	flag.Parse()
	if *fnPtr == "createNewServer" {
		droplet := createDroplet(client)
		waitUntilDropletReady(client, droplet.ID)
		droplet, _, _ = client.Droplets.Get(oauth2.NoContext, droplet.ID)

		ipv4, _ := droplet.PublicIPv4()
		addr := doDropletInfoSite + strconv.Itoa(droplet.ID)
		fmt.Printf("%s: %s @%s\n", ipv4, droplet.Name, addr)

		// Write /tmp/new_digital_ocean_droplet_params
		envVarsFile := []byte("export NEW_SERVER_IPV4=" + ipv4 + "\nexport NEW_DROPLET_ID=" + strconv.Itoa(droplet.ID) + "\n")
		err := ioutil.WriteFile("/tmp/new_digital_ocean_droplet_params", envVarsFile, 0644)
		if err != nil {
			fmt.Printf("Failed to write /tmp/new_digital_ocean_droplet_params: %s", err)
		}

		var firewallID = os.Getenv("CTX_DIGITALOCEAN_FIREWALL")
		_, err2 := client.Firewalls.AddDroplets(oauth2.NoContext, firewallID, droplet.ID)
		if err2 != nil {
			fmt.Printf("Failed to add droplet to Firewall: %s", err2)
		}
	} else if *fnPtr == "deleteServer" {
		dropletIDPtr := flag.String("dropletID", "<digitalOceanDropletID>", "DO droplet to attach floatingIP to")

		dropletID, _ := strconv.Atoi(*dropletIDPtr)
		droplet, _, _ := client.Droplets.Get(oauth2.NoContext, dropletID)
		fmt.Printf("\ndeleting DropletID: %d\n", droplet.ID)

		common.DeleteDODroplet(dropletID)
	} else if *fnPtr == "toggleSSHFirewall" {
		ipAddrPtr := flag.String("ipAddr", "<ipAddress to add>", "so deploying agent can access Droplet")
		ipSubrPtr := flag.String("ipSubr", "<ipAddress to remove>", "deploying agent no longer needs access")

		if *ipAddrPtr != "" {
			common.ToggleSSHipAddress(true, getCurrentIP(), client)
		} else if *ipSubrPtr != "" {
			common.ToggleSSHipAddress(false, getCurrentIP(), client)
		}
	}
}

func getCurrentIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func updateIPV6(client *godo.Client, ipv6 string, hostname string, domainID int) {
	record, _, err := client.Domains.Record(oauth2.NoContext, hostname, domainID)
	if err != nil {
		log.Printf("unable to updateIPv6 for %s: %s", hostname, err.Error())
	}
	fmt.Printf("current IPv6 %s: %s => %s\n", record.Name, record.Type, record.Data)

	editRequest := &godo.DomainRecordEditRequest{
		Type: record.Type,
		Name: record.Name,
		Data: strings.ToLower(ipv6),
	}
	_, _, err = client.Domains.EditRecord(oauth2.NoContext, hostname, domainID, editRequest)
	for err != nil {
		fmt.Printf("FAIL domain update IPv6: %s\n", err)
		time.Sleep(5 * time.Second)
		_, _, err = client.Domains.EditRecord(oauth2.NoContext, hostname, domainID, editRequest)
	}
}

// wait for in-progress actions to complete
func waitUntilDropletReady(client *godo.Client, dropletID int) {
	opt := &godo.ListOptions{}

	for ready := false; !ready; {
		actions, _, _ := client.Droplets.Actions(oauth2.NoContext, dropletID, opt)
		ready = true
		for j, action := range actions {
			fmt.Printf("%d: %s => %s\n", j, action.Type, action.Status)
			if action.Status == "in-progress" {
				ready = false
			}
		}
		if !ready {
			time.Sleep(time.Second * 5)
		}
	}
}

func reassignFloatingIP(client *godo.Client, droplet *godo.Droplet) {
	client.FloatingIPActions.Unassign(oauth2.NoContext, common.FloatingIPAddress)

	_, _, err := client.FloatingIPActions.Assign(oauth2.NoContext, common.FloatingIPAddress, droplet.ID)
	for err != nil {
		fmt.Printf("WARN: %s\n", err.Error())
		time.Sleep(5 * time.Second)
		_, _, err = client.FloatingIPActions.Assign(oauth2.NoContext, common.FloatingIPAddress, droplet.ID)
	}
}

func createDroplet(client *godo.Client) *godo.Droplet {
	var newDroplet *godo.Droplet

	fingerprint := os.Getenv("CTX_SSH_DEPLOY_FINGERPRINT")
	dropletName := "b" + githubBuild + ".ackerson.de"

	sshKeys := []godo.DropletCreateSSHKey{}
	sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: fingerprint})

	digitaloceanIgnitionJSON, err := ioutil.ReadFile("do_ubuntu_userdata.sh")
	if err != nil {
		fmt.Printf("Failed to read JSON file: %s", err)
	} else {
		createRequest := &godo.DropletCreateRequest{
			Name:   dropletName,
			Region: "fra1",
			Size:   "s-1vcpu-1gb-amd", //ubuntu-s-1vcpu-1gb-amd-fra1-01
			Image: godo.DropletCreateImage{
				Slug: "ubuntu-20-10-x64",
			},
			IPv6:     true,
			SSHKeys:  sshKeys,
			UserData: string(digitaloceanIgnitionJSON),
		}

		newDroplet, _, err = client.Droplets.Create(oauth2.NoContext, createRequest)
		if err != nil {
			fmt.Printf("\nUnexpected ERROR: %s\n\n", err)
			os.Exit(1)
		}
	}

	return newDroplet
}
