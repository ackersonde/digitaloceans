#!/bin/bash
cat > /root/.ssh/id_rsa <<EOF
$CTX_RASPBERRYPI_SSH_PRIVKEY
EOF

chmod 400 /root/.ssh/id_rsa
