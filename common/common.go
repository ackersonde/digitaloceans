package common

import (
	"bufio"
	"context"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
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

var doPersonalAccessToken = os.Getenv("CTX_DIGITALOCEAN_TOKEN")
var firewallID = os.Getenv("CTX_DIGITALOCEAN_FIREWALL")

// FloatingIPAddress is the static IP for ackerson.de
var FloatingIPAddress = os.Getenv("doFloatingIP")

// AccessDigitalOceanSpaces returns an S3 client for DO Spaces work
func AccessDigitalOceanSpaces() *minio.Client {
	accessKey := os.Getenv("CTX_SPACES_KEY")
	secKey := os.Getenv("CTX_SPACES_SECRET")
	endpoint := "ams3.digitaloceanspaces.com"
	ssl := true

	// Initiate a client using DigitalOcean Spaces.
	client, err := minio.New(endpoint, accessKey, secKey, ssl)
	if err != nil {
		log.Fatal(err)
	}

	return client
}

// PrepareDigitalOceanLogin does what it says on the box
func PrepareDigitalOceanLogin() *godo.Client {
	tokenSource := &TokenSource{
		AccessToken: doPersonalAccessToken,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return godo.NewClient(oauthClient)
}

func prepareSSHipAddresses() []string {
	ipAddys := []string{}
	ipAddrs, _ := net.LookupIP(os.Getenv("homeDomain"))
	for _, ipAddr := range ipAddrs {
		ipAddys = append(ipAddys, ipAddr.String())
	}

	// whitelist UptimeRobot addys
	uptimeRobotAddresses, err := urlToLines("https://uptimerobot.com/inc/files/ips/IPv4andIPv6.txt")
	if err != nil {
		log.Println(err.Error())
	}
	ipAddys = append(ipAddys, uptimeRobotAddresses...)

	return ipAddys
}

func urlToLines(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return linesFromReader(resp.Body)
}

func linesFromReader(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// ToggleSSHipAddress adds/removes an IP address on the FW rule
func ToggleSSHipAddress(add bool, ipAddress string, client *godo.Client) {
	ctx := context.TODO()

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

// UpdateFirewall to maintain connectivity while Telekom rotates IPs
func UpdateFirewall() {
	ipAddys := prepareSSHipAddresses()

	client := PrepareDigitalOceanLogin()
	ctx := context.TODO()

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

// DeleteDODroplet more here https://developers.digitalocean.com/documentation/v2/#delete-a-droplet
func DeleteDODroplet(ID int) string {
	var result string

	client := PrepareDigitalOceanLogin()

	_, err := client.Droplets.Delete(oauth2.NoContext, ID)
	if err == nil {
		result = "Successfully deleted Droplet `" + strconv.Itoa(ID) + "`"
	} else {
		result = err.Error()
	}

	return result
}

// CopyFileToDOSpaces is a helper func for copying files to DigitalOcean Spaces
// Helpful ideas: https://github.com/minio/minio-go/tree/master/examples/s3
func CopyFileToDOSpaces(spacesName string, remoteFile string, url string, filesize int64) (err error) {
	var reader io.Reader

	resp, err := http.Get(url)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported protocol scheme \"\"") {
			reader, err = os.Open(url)
			if err != nil {
				log.Printf("can't find local file %s: %s", url, err.Error())
			}
		}
	} else {
		reader = bufio.NewReader(resp.Body)
	}

	remoteFile = strings.TrimPrefix(remoteFile, "/")
	mimeType := mime.TypeByExtension(filepath.Ext(remoteFile))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	doSpacesClient := AccessDigitalOceanSpaces()

	userMetaData := map[string]string{"x-amz-acl": "public-read"}
	wrote, err := doSpacesClient.PutObject(spacesName, remoteFile, reader, filesize,
		minio.PutObjectOptions{UserMetadata: userMetaData, ContentType: mimeType})
	if err != nil {
		return err
	}

	log.Printf("successfully wrote %d bytes to DO Spaces (%s)\n", wrote, remoteFile)

	return nil
}

// DownloadFromDOSpaces is a helper function for downloading files from DOS
func DownloadFromDOSpaces(spacesName string, w http.ResponseWriter, r *http.Request) {
	minioClient := AccessDigitalOceanSpaces()

	vars := mux.Vars(r)
	fileName := vars["file"]

	if fileName == "" {
		http.Error(w, "Listing not allowed", http.StatusUnauthorized)
		return
	}
	file, err := minioClient.GetObject(spacesName, fileName, minio.GetObjectOptions{})
	if err != nil {
		http.Error(w, "Couldn't find '"+fileName+"'", http.StatusBadRequest)
		return
	}

	if _, err = io.Copy(w, file); err != nil {
		http.Error(w, "Couldn't write "+spacesName+"/"+fileName, http.StatusBadRequest)
		return
	}
}
