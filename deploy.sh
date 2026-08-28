#!/bin/zsh
set -e
cd "$(dirname "$0")"
if [[ ! -f ./config.json ]]; then
  echo "missing ./config.json, copy it here first" >&2
  exit 1
fi
pkill simpleserver 2>/dev/null || true
nohup ./simpleserver -c ./config.json >>1.txt 2>>2.txt &!
echo "simpleserver started with ./config.json"
