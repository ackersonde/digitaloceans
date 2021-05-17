package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ackersonde/digitaloceans/common"
	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
)

var githubBuild = os.Getenv("GITHUB_RUN_ID")
var envFile = "/tmp/new_digital_ocean_droplet_params"

func main() {
	client := common.PrepareDigitalOceanLogin()

	fnPtr := flag.String("fn", "createNewServer|deleteServer|firewallSSH", "which function to run")
	dropletIDPtr := flag.String("dropletID", "<digitalOceanDropletID>", "DO droplet to attach floatingIP to")
	allowPtr := flag.Bool("allow", false, "so deploying agent can access Droplet")
	ipPtr := flag.String("ip", "<internet ip addr of github action instance>", "see prev param")
	tagPtr := flag.String("tag", "dynamic", "tag to add to droplet")
	flag.Parse()

	if *fnPtr == "createNewServer" {
		existingDeployDroplet := findExistingDeployDroplet(client, *tagPtr)
		existingIPv6, _ := existingDeployDroplet.PublicIPv6()

		droplet := createDroplet(client, *tagPtr)
		waitUntilDropletReady(client, droplet.ID)

		droplet, _, _ = client.Droplets.Get(context.Background(), droplet.ID)
		ipv4, _ := droplet.PublicIPv4()
		ipv6, _ := droplet.PublicIPv6()

		// Write /tmp/new_digital_ocean_droplet_params
		envVarsFile := []byte(
			"export NEW_SERVER_IPV4=" + ipv4 +
				"\nexport NEW_SERVER_IPV6=" + ipv6 +
				"\nexport NEW_DROPLET_ID=" + strconv.Itoa(droplet.ID) +
				"\nexport OLD_DROPLET_ID=" + strconv.Itoa(existingDeployDroplet.ID) +
				"\nexport OLD_SERVER_IPV6=" + existingIPv6)

		err := ioutil.WriteFile(envFile, envVarsFile, 0644)
		if err != nil {
			fmt.Printf("Failed to write %s: %s", envFile, err)
		}

		var firewallID = os.Getenv("CTX_DIGITALOCEAN_FIREWALL")
		_, err2 := client.Firewalls.AddDroplets(context.Background(), firewallID, droplet.ID)
		if err2 != nil {
			fmt.Printf("Failed to add droplet to Firewall: %s", err2)
		}
	} else if *fnPtr == "deleteServer" {
		dropletID, _ := strconv.Atoi(*dropletIDPtr)
		droplet, _, _ := client.Droplets.Get(context.Background(), dropletID)

		client.Droplets.Delete(context.Background(), dropletID)

		fmt.Printf("\ndeleted DropletID: %d\n", droplet.ID)
	} else if *fnPtr == "firewallSSH" {
		common.ToggleSSHipAddress(*allowPtr, *ipPtr, client)
	} else if *fnPtr == "updateDNS" {
		dropletID, _ := strconv.Atoi(*dropletIDPtr)
		droplet, _, _ := client.Droplets.Get(context.Background(), dropletID)
		ipv4, _ := droplet.PublicIPv4()
		ipv6, _ := droplet.PublicIPv6()

		updateDNS(client, ipv6, "ackerson.de", 23738236)
		updateDNS(client, ipv4, "ackerson.de", 23738257)
	}
}

func findExistingDeployDroplet(client *godo.Client, tag string) godo.Droplet {
	var droplet godo.Droplet
	droplet.ID = 1 // set default, nonsensical value
	if tag == "traefik" {
		droplets, _, _ := client.Droplets.ListByTag(context.Background(), tag, &godo.ListOptions{})
		if len(droplets) > 0 {
			droplet = droplets[0]
		}
	}

	return droplet
}

func updateDNS(client *godo.Client, ipAddr string, hostname string, domainID int) {
	record, _, err := client.Domains.Record(context.Background(), hostname, domainID)
	if err != nil {
		log.Printf("unable to updateDNS for %s: %s", hostname, err.Error())
	}
	fmt.Printf("current DNS %s: %s => %s\n", record.Name, record.Type, record.Data)

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

// wait for in-progress actions to complete
func waitUntilDropletReady(client *godo.Client, dropletID int) {
	opt := &godo.ListOptions{}
	j := 0

	for ready := false; !ready; {
		actions, _, _ := client.Droplets.Actions(context.Background(), dropletID, opt)
		ready = true
		for _, action := range actions {
			fmt.Printf("%d: %s => %s\n", j, action.Type, action.Status)
			if action.Status == "in-progress" {
				ready = false
				j++
				break
			}
		}
		if !ready {
			time.Sleep(time.Second * 5)
		}
	}
}

func reassignFloatingIP(client *godo.Client, droplet *godo.Droplet) {
	client.FloatingIPActions.Unassign(context.Background(), common.FloatingIPAddress)

	_, _, err := client.FloatingIPActions.Assign(context.Background(), common.FloatingIPAddress, droplet.ID)
	for err != nil {
		fmt.Printf("WARN: %s\n", err.Error())
		time.Sleep(5 * time.Second)
		_, _, err = client.FloatingIPActions.Assign(context.Background(), common.FloatingIPAddress, droplet.ID)
	}
}

func createSSHKey(client *godo.Client) *godo.Key {
	privateKeyPair, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Printf("rsa.GenerateKey returned error: %v", err)
	}

	publicRsaKey, err := ssh.NewPublicKey(privateKeyPair.Public())
	if err != nil {
		log.Printf("ssh.NewPublicKey returned error: %v", err)
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	createRequest := &godo.KeyCreateRequest{
		Name:      githubBuild + "SSHkey",
		PublicKey: string(pubKeyBytes),
	}

	key, _, err := client.Keys.Create(context.Background(), createRequest)
	if err != nil {
		log.Printf("Keys.Create returned error: %v", err)
	} else {
		pemdata := pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKeyPair),
			},
		)
		err := ioutil.WriteFile("/home/runner/.ssh/id_rsa", pemdata, 0400)
		if err != nil {
			fmt.Printf("Failed to write /home/runner/.ssh/id_rsa: %s", err)
		}
	}

	return key
}

func createDroplet(client *godo.Client, tag string) *godo.Droplet {
	var newDroplet *godo.Droplet

	fingerprint := os.Getenv("CTX_SSH_DEPLOY_FINGERPRINT")
	dropletName := "b" + githubBuild + ".ackerson.de"

	sshKeys := []godo.DropletCreateSSHKey{}
	sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: fingerprint})

	digitaloceanIgnitionJSON, err := ioutil.ReadFile("digitalocean_ubuntu_userdata.sh")
	if err != nil {
		fmt.Printf("Failed to read JSON file: %s", err)
	} else {
		if tag == "" {
			tag = "dynamic"
		}
		createRequest := &godo.DropletCreateRequest{
			Name:   dropletName,
			Region: "fra1",
			Size:   "s-1vcpu-1gb-amd",
			Image: godo.DropletCreateImage{
				Slug: "ubuntu-20-10-x64",
			},
			IPv6:       true,
			Monitoring: true,
			SSHKeys:    sshKeys,
			UserData:   string(digitaloceanIgnitionJSON),
			Tags:       []string{tag},
		}

		newDroplet, _, err = client.Droplets.Create(context.Background(), createRequest)
		if err != nil {
			fmt.Printf("\nUnexpected ERROR: %s\n\n", err)
			os.Exit(1)
		}
	}

	return newDroplet
}
