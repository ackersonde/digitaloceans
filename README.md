[<img src="https://jvandyke.gallerycdn.vsassets.io/extensions/jvandyke/vscode-circleci/0.1.2/1474455189849/Microsoft.VisualStudio.Services.Icons.Default" width="16">![CircleCI](https://circleci.com/gh/danackerson/digitalocean.svg?style=svg)](https://circleci.com/gh/danackerson/digitalocean)

# Build & Deploy [DigitalOcean Droplet](https://cloud.digitalocean.com/droplets)
Using the golang api from [godo](https://www.digitalocean.com/community/projects/godo), this CircleCI build creates a [custom](https://github.com/danackerson/digitalocean/blob/master/digitalocean_ignition.json) CoreOS <img src="https://coreos.com/assets/ico/favicon.png" width="16"> droplet in NYC3.

Following Environment variables need to be set in your [CircleCI project](https://circleci.com/gh/danackerson/digitalocean/edit#env-vars):
* deployUser
* doPersonalAccessToken
* encodedCircleCIDeployPubKey
* encodedConsolePasswdHash
* encodedDOSSHLoginPubKey

# WARNING
Every push to this repo will result in a new Droplet created at DigitalOcean => +$5 / month!

Use git commit msg string snippet `[ci skip]` to avoid this!
