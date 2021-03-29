#!/bin/bash

KEY_FILE_NAME=/root/.ssh/id_rsa

cat > "$KEY_FILE_NAME" <<EOF
{{CTX_RASPBERRYPI_SSH_PRIVKEY}}
EOF

chmod 400 "$KEY_FILE_NAME"

