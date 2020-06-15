#!/usr/bin/env bash

export ELRONDTESTNETSCRIPTSDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source "$ELRONDTESTNETSCRIPTSDIR/variables.sh"
source "$ELRONDTESTNETSCRIPTSDIR/include/config.sh"

# Load local overrides, .gitignored
LOCAL_OVERRIDES="$ELRONDTESTNETSCRIPTSDIR/local.sh"
if [ -f "$LOCAL_OVERRIDES" ]; then
  source "$ELRONDTESTNETSCRIPTSDIR/local.sh"
fi

VALIDATOR_RES_PORT="$PORT_ORIGIN_VALIDATOR_REST"

if [ -z "$1" ]; then
  echo "epoch argument was not provided. Usage: './hardfork.sh [epoch number]' as in './hardfork.sh 1'"
  exit
fi

if [ $1 -lt "1" ]; then
  echo "incorrect epoch argument was provided. Usage: './hardfork.sh [epoch number]' as in './hardfork.sh 1'"
  exit
fi

epoch=$1
cmd=(printf "$(curl -d '{"epoch":'"$epoch"'}' -H 'Content-Type: application/json' http://127.0.0.1:$VALIDATOR_RES_PORT/hardfork/trigger)")
"${cmd[@]}"

echo " done curl"

# change the setting from config.toml: AfterHardFork to true
updateTOMLValue "$TESTNETDIR/node/config/config_validator.toml" "AfterHardFork" "true"
updateTOMLValue "$TESTNETDIR/node/config/config_observer.toml" "AfterHardFork" "true"

# change nodesSetup.json genesis time to a new value
let startTime="$(date +%s) + $HARDFORK_DELAY"
updateJSONValue "$TESTNETDIR/node/config/nodesSetup.json" "startTime" "$startTime"

# copy back the configs
if [ $COPY_BACK_CONFIGS -eq 1 ]; then
  copyBackConfigs
fi

echo "done hardfork reconfig"
