package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danackerson/digitalocean/common"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var doDropletInfoSite = "https://cloud.digitalocean.com/droplets/"
var encodedDOSSHLoginPubKey = os.Getenv("encodedDOSSHLoginPubKey")
var doPersonalAccessToken = os.Getenv("doPersonalAccessToken")
var circleCIBuild = os.Getenv("CIRCLE_BUILD_NUM")

func main() {
	client := common.PrepareDigitalOceanLogin()

	fnPtr := flag.String("fn", "updateDNS|createNewServer", "which function to run")
	dropletIDPtr := flag.String("dropletID", "<digitalOceanDropletID>", "DO droplet to attach floatingIP to")
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
	} else {
		dropletID, _ := strconv.Atoi(*dropletIDPtr)
		droplet, _, _ := client.Droplets.Get(oauth2.NoContext, dropletID)
		fmt.Printf("\ngoing to work on DropletID: %d\n", droplet.ID)

		reassignFloatingIP(client, droplet)
		common.UpdateFirewall()

		// update ipv6 DNS entry to new droplet
		ipv6, _ := droplet.PublicIPv6()
		fmt.Printf("new IPv6 addr: %s\n", ipv6)
		ackersonDERecordIDIPv6 := 23738236
		record, _, _ := client.Domains.Record(oauth2.NoContext, "ackerson.de", ackersonDERecordIDIPv6)
		fmt.Printf("current IPv6 %s: %s => %s", record.Name, record.Type, record.Data)

		editRequest := &godo.DomainRecordEditRequest{
			Type: record.Type,
			Name: record.Name,
			Data: strings.ToLower(ipv6),
		}
		_, _, err := client.Domains.EditRecord(oauth2.NoContext, "ackerson.de", ackersonDERecordIDIPv6, editRequest)
		for err != nil {
			fmt.Printf("FAIL domain update IPv6: %s\n", err)
			_, _, err = client.Domains.EditRecord(oauth2.NoContext, "ackerson.de", ackersonDERecordIDIPv6, editRequest)
		}
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
	floatingIPAddress := os.Getenv("floatingIPAddress")
	client.FloatingIPActions.Unassign(oauth2.NoContext, floatingIPAddress)

	_, _, err := client.FloatingIPActions.Assign(oauth2.NoContext, floatingIPAddress, droplet.ID)
	for err != nil {
		fmt.Printf("WARN: %s\n", err.Error())
		_, _, err = client.FloatingIPActions.Assign(oauth2.NoContext, floatingIPAddress, droplet.ID)
	}
}

func createDroplet(client *godo.Client) *godo.Droplet {
	var newDroplet *godo.Droplet

	fingerprint := "e0:a3:4c:5a:5a:1b:9c:bb:b5:51:a7:7f:62:27:51:96"
	dropletName := "b" + circleCIBuild + ".ackerson.de"

	sshKeys := []godo.DropletCreateSSHKey{}
	sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: fingerprint})

	digitaloceanIgnitionJSON, err := ioutil.ReadFile("digitalocean_ignition.json")
	if err != nil {
		fmt.Printf("Failed to read JSON file: %s", err)
	} else {
		createRequest := &godo.DropletCreateRequest{
			Name:   dropletName,
			Region: "fra1",
			Size:   "512mb",
			Image: godo.DropletCreateImage{
				Slug: "coreos-stable",
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
