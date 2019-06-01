#!/bin/sh

if [ -f doFirewall.json ]; then
  # Add the droplet back to the original Firewall
  echo '{"name":' > reenableFW.json
  cat doFirewall.json | jq '.firewall.name' >> reenableFW.json && echo ', "inbound_rules":' >> reenableFW.json
  cat doFirewall.json | jq '.firewall.inbound_rules' >> reenableFW.json && echo ', "outbound_rules":' >> reenableFW.json
  cat doFirewall.json | jq '.firewall.outbound_rules' >> reenableFW.json && echo ', "droplet_ids":' >> reenableFW.json
  cat doFirewall.json | jq '.firewall.droplet_ids' >> reenableFW.json
  echo "}" >> reenableFW.json

  # Apply reenabledFW rules
  curl -X PUT -H "Content-Type: application/json" -d @reenableFW.json -H "Authorization: Bearer $CTX_DIGITALOCEAN_TOKEN" "https://api.digitalocean.com/v2/firewalls/$CTX_DIGITALOCEAN_FIREWALL"
else
  echo 'doFirewall.json not found -> exiting...'
fi
