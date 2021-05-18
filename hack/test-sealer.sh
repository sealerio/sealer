#!/bin/bash
SEALER_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

# run test
echo "starting to test sealer ..."
cd $SEALER_ROOT/test && go test