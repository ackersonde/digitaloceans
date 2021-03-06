package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ackersonde/digitaloceans/common"
	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
)

var sshPrivateKeyFilePath = "/home/runner/.ssh/id_rsa"
var githubBuild = os.Getenv("GITHUB_RUN_ID")
var envFile = "/tmp/new_digital_ocean_droplet_params"

func main() {
	client := common.PrepareDigitalOceanLogin()

	fnPtr := flag.String("fn", "createNewServer|deleteServer|firewallSSH|deleteSSHKey", "which function to run")
	dropletIDPtr := flag.String("dropletID", "<digitalOceanDropletID>", "DO droplet to attach floatingIP to")
	sshKeyPtr := flag.String("sshKeyID", "<digitalOceanSSHKeyID>", "DO ssh key ID")
	allowPtr := flag.Bool("allow", false, "so deploying agent can access Droplet")
	ipPtr := flag.String("ip", "<internet ip addr of github action instance>", "see prev param")
	tagPtr := flag.String("tag", "dynamic", "tag to add to droplet")
	flag.Parse()

	if *fnPtr == "createNewServer" {
		existingDeployDroplet := common.FindExistingDeployDroplet(*tagPtr)
		existingIPv6, _ := existingDeployDroplet.PublicIPv6()

		droplet, sshKeyID := createDroplet(client, *tagPtr)
		waitUntilDropletReady(client, droplet.ID)

		// now that Droplet is READY, get IP addresses
		droplet, _, _ = client.Droplets.Get(context.Background(), droplet.ID)

		// IP addresses aren't immediately available, so wait until you get them
		ipv4, err := droplet.PublicIPv4()
		for err != nil || ipv4 == "" {
			if err == nil {
				err = errors.New("no ipv4 yet")
			}
			fmt.Printf("IPv4 fail: %s\n", err.Error())
			time.Sleep(time.Second * 5)
			droplet, _, _ = client.Droplets.Get(context.Background(), droplet.ID)
			ipv4, err = droplet.PublicIPv4()
		}
		ipv6, err2 := droplet.PublicIPv6()
		for err2 != nil || ipv6 == "" {
			if err2 == nil {
				err2 = errors.New("no ipv6 yet")
			}
			fmt.Printf("IPv6 fail: %s\n", err2.Error())
			time.Sleep(time.Second * 5)
			droplet, _, _ = client.Droplets.Get(context.Background(), droplet.ID)
			ipv6, err2 = droplet.PublicIPv6()
		}

		// Write /tmp/new_digital_ocean_droplet_params
		envVarsFile := []byte(
			"export NEW_SERVER_IPV4=" + ipv4 +
				"\nexport NEW_SERVER_IPV6=" + ipv6 +
				"\nexport DEPLOY_KEY_ID=" + sshKeyID +
				"\nexport NEW_DROPLET_ID=" + strconv.Itoa(droplet.ID) +
				"\nexport OLD_DROPLET_ID=" + strconv.Itoa(existingDeployDroplet.ID) +
				"\nexport OLD_SERVER_IPV6=" + existingIPv6)

		err = ioutil.WriteFile(envFile, envVarsFile, 0644)
		if err != nil {
			fmt.Printf("Failed to write %s: %s", envFile, err)
		}

		var firewallID = os.Getenv("CTX_DIGITALOCEAN_FIREWALL")
		_, err3 := client.Firewalls.AddDroplets(context.Background(), firewallID, droplet.ID)
		if err3 != nil {
			fmt.Printf("Failed to add droplet to Firewall: %s", err3)
		}
	} else if *fnPtr == "deleteServer" {
		dropletID, _ := strconv.Atoi(*dropletIDPtr)
		droplet, _, _ := client.Droplets.Get(context.Background(), dropletID)

		client.Droplets.Delete(context.Background(), dropletID)

		fmt.Printf("\ndeleted DropletID: %d\n", droplet.ID)
	} else if *fnPtr == "firewallSSH" {
		common.ToggleSSHipAddress(*allowPtr, *ipPtr, client)
	} else if *fnPtr == "deleteSSHKey" {
		sshKeyID, err := strconv.Atoi(*sshKeyPtr)
		if err != nil {
			fmt.Printf("Failed to convert sshKeyID: %s", err)
		} else {
			common.DeleteSSHKey(sshKeyID, client)
		}
	} else if *fnPtr == "updateDNS" {
		dropletID, _ := strconv.Atoi(*dropletIDPtr)
		droplet, _, _ := client.Droplets.Get(context.Background(), dropletID)
		ipv4, _ := droplet.PublicIPv4()
		ipv6, _ := droplet.PublicIPv6()

		common.UpdateDNSentry(ipv6, "ackerson.de", 23738236)
		common.UpdateDNSentry(ipv4, "ackerson.de", 23738257)
		common.UpdateDNSentry(ipv6, "hausmeisterservice-planb.de", 302721441)
		common.UpdateDNSentry(ipv4, "hausmeisterservice-planb.de", 302721419)
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
		err := ioutil.WriteFile(sshPrivateKeyFilePath, pemdata, 0400)
		if err != nil {
			fmt.Printf("Failed to write %s: %s", sshPrivateKeyFilePath, err.Error())
		}
	}

	return key
}

func createDroplet(client *godo.Client, tag string) (*godo.Droplet, string) {
	var newDroplet *godo.Droplet

	deploymentKey := createSSHKey(client)
	dropletName := "b" + githubBuild + ".ackerson.de"

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
				Slug: "ubuntu-22-04-x64",
			},
			IPv6:       true,
			Monitoring: true,
			SSHKeys:    []godo.DropletCreateSSHKey{{Fingerprint: deploymentKey.Fingerprint}},
			UserData:   string(digitaloceanIgnitionJSON),
			Tags:       []string{tag},
		}

		newDroplet, _, err = client.Droplets.Create(context.Background(), createRequest)
		if err != nil {
			fmt.Printf("\nUnexpected ERROR: %s\n\n", err)
			fmt.Printf("Deleted ssh key: %s", common.DeleteSSHKey(deploymentKey.ID, client))
		}
	}

	return newDroplet, strconv.Itoa(deploymentKey.ID)
}
