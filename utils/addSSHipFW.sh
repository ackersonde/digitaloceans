#!/bin/bash
json='{"inbound_rules":[{"protocol":"tcp","ports":"22","sources":{"addresses":["'"$public_ip_address"'"]}}]}'

curl -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $digitalOceanToken" \
-d $json "https://api.digitalocean.com/v2/firewalls/$doFirewallID/rules"
