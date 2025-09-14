#!/bin/bash

ssh "contabo-01" << EOF
  set -e
  cd de.ohrenpirat.lasttesttest
  docker compose pull
  docker compose up -d
EOF