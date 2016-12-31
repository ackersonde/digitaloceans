package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

var doDropletInfoSite = "https://cloud.digitalocean.com/droplets/"

var encodedDOSSHLoginPubKey = os.Getenv("encodedDOSSHLoginPubKey")
var doPersonalAccessToken = os.Getenv("doPersonalAccessToken")
var circleCIBuild = os.Getenv("CIRCLE_BUILD_NUM")

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

func main() {
	tokenSource := &TokenSource{
		AccessToken: doPersonalAccessToken,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplets, err := DropletList(client)
	if err != nil {
		fmt.Printf("Failed to list droplets: %s", err)
	}
	for _, droplet := range droplets {
		ipv4, _ := droplet.PublicIPv4()
		addr := doDropletInfoSite + strconv.Itoa(droplet.ID)
		fmt.Printf("%s: %s @%s\n", ipv4, droplet.Name, addr)
	}

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

		//fmt.Printf("Droplet creation request: %v", createRequest)

		newDroplet, _, err := client.Droplets.Create(createRequest)
		if err != nil {
			fmt.Printf("\nUnexpected ERROR: %s\n\n", err)
			// TODO: set exit code to !0 (so CircleCI reports failed build)
			os.Exit(1)
		} else {
			ipv4, _ := newDroplet.PublicIPv4()
			addr := doDropletInfoSite + strconv.Itoa(newDroplet.ID)
			fmt.Printf("\n%s: %s CREATED @ %s\n", ipv4, newDroplet.Name, addr)
			os.Exit(0)
		}
	}
}

// DropletList is now commented
func DropletList(client *godo.Client) ([]godo.Droplet, error) {
	// create a list to hold our droplets
	list := []godo.Droplet{}

	// create options. initially, these will be blank
	opt := &godo.ListOptions{}
	for {
		droplets, resp, err := client.Droplets.List(opt)
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
