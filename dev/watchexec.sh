#!/bin/env sh
args=$@
cwd=$(pwd)
echo "Starting watchexec on ${cwd}"

set -x
export LOG_LEVEL=info
watchexec -n -q -r -e go,mod,sum -- sh -c "while true; do sleep .2 && go run main.go ${args}; done"