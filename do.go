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
	keyIDPtr := flag.String("keyID", "<sshKeyID>", "DO droplet sshKey")
	allowPtr := flag.Bool("allow", false, "so deploying agent can access Droplet")
	ipPtr := flag.String("ip", "<internet ip addr of github action instance>", "see prev param")
	tagPtr := flag.String("tag", "<tag>", "tag to add to droplet")
	flag.Parse()

	if *fnPtr == "createNewServer" {
		key := createSSHKey(client)

		droplet := createDroplet(client, key)
		waitUntilDropletReady(client, droplet.ID)

		var oldDroplet godo.Droplet
		droplet, _, _ = client.Droplets.Get(context.Background(), droplet.ID)
		if *tagPtr != "" {
			oldDroplets, _, _ := client.Droplets.ListByTag(context.Background(), *tagPtr, &godo.ListOptions{})
			if len(oldDroplets) > 0 {
				oldDroplet = oldDroplets[0]
			}
			droplet.Tags = append(droplet.Tags, *tagPtr)
		}

		ipv4, _ := droplet.PublicIPv4()
		ipv6, _ := droplet.PublicIPv6()

		// Write /tmp/new_digital_ocean_droplet_params
		envVarsFile := []byte(
			"export NEW_SERVER_IPV4=" + ipv4 +
				"\nexport NEW_SERVER_IPV6=" + ipv6 +
				"\nexport NEW_DROPLET_ID=" + strconv.Itoa(droplet.ID) +
				"\nexport OLD_DROPLET_ID=" + strconv.Itoa(oldDroplet.ID) +
				"\nexport NEW_SSH_KEY_ID=" + strconv.Itoa(key.ID))

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
		keyID, _ := strconv.Atoi(*keyIDPtr)
		droplet, _, _ := client.Droplets.Get(context.Background(), dropletID)
		fmt.Printf("\ndeleting DropletID: %d\n", droplet.ID)

		resp, _ := client.Keys.DeleteByID(context.Background(), keyID)
		fmt.Printf("Keys.DeleteByID? %s", resp)
		resp, _ = client.Droplets.Delete(context.Background(), dropletID)
		fmt.Printf("Droplets.Delete? %s", resp)
	} else if *fnPtr == "firewallSSH" {
		common.ToggleSSHipAddress(*allowPtr, *ipPtr, client)
		if !*allowPtr {
			keyID, _ := strconv.Atoi(*keyIDPtr)
			resp, _ := client.Keys.DeleteByID(context.Background(), keyID)
			fmt.Printf("Keys.DeleteByID? %s", resp)
		} else {
			_, err := os.Stat(envFile)
			if os.IsNotExist(err) {
				key := createSSHKey(client)
				// Write /tmp/new_digital_ocean_droplet_params
				envVarsFile := []byte(
					"export NEW_SSH_KEY_ID=" + strconv.Itoa(key.ID))

				err := ioutil.WriteFile(envFile, envVarsFile, 0644)
				if err != nil {
					fmt.Printf("Failed to write %s: %s", envFile, err)
				}
			}
		}
	} else if *fnPtr == "updateDNS" {
		dropletID, _ := strconv.Atoi(*dropletIDPtr)
		droplet, _, _ := client.Droplets.Get(context.Background(), dropletID)
		ipv4, _ := droplet.PublicIPv4()
		ipv6, _ := droplet.PublicIPv6()

		updateDNS(client, ipv6, "ackerson.de", 23738236)
		updateDNS(client, ipv4, "ackerson.de", 23738257)
	}
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

	log.Println("Public key generated")

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

func createDroplet(client *godo.Client, key *godo.Key) *godo.Droplet {
	var newDroplet *godo.Droplet

	fingerprint := os.Getenv("CTX_SSH_DEPLOY_FINGERPRINT")
	dropletName := "b" + githubBuild + ".ackerson.de"

	sshKeys := []godo.DropletCreateSSHKey{}
	sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: fingerprint})
	sshKeys = append(sshKeys, godo.DropletCreateSSHKey{Fingerprint: key.Fingerprint})

	digitaloceanIgnitionJSON, err := ioutil.ReadFile("digitalocean_ubuntu_userdata.sh")
	if err != nil {
		fmt.Printf("Failed to read JSON file: %s", err)
	} else {
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
		}

		newDroplet, _, err = client.Droplets.Create(context.Background(), createRequest)
		if err != nil {
			fmt.Printf("\nUnexpected ERROR: %s\n\n", err)
			os.Exit(1)
		}
	}

	return newDroplet
}
