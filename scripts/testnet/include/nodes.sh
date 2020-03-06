source "$ELRONDTESTNETSCRIPTSDIR/variables.sh"
source "$ELRONDTESTNETSCRIPTSDIR/include/terminal.sh"

startSeednode() {
  setTerminalSession "elrond-tools"
  setTerminalLayout "even-horizontal"

  setWorkdirForNextCommands "$TESTNETDIR/seednode"
  
  if [ -n "$NODE_NICENESS" ]
  then
    seednodeCommand="nice -n $NODE_NICENESS ./seednode"
  else
    seednodeCommand="./seednode"
  fi

  runCommandInTerminal "$seednodeCommand" $1 v
}

stopSeednode() {
  stopProcessByPort $PORT_SEEDNODE
}

startObservers() {
  setTerminalSession "elrond-nodes"
  setTerminalLayout "tiled"

  setWorkdirForNextCommands "$TESTNETDIR/node"

  OBSERVER_INDEX=0
  # Start Shard Observers
  let "max_shard_id=$SHARDCOUNT - 1"
  for SHARD in `seq 0 1 $max_shard_id`; do
    for OBSERVER_IN_SHARD in `seq $SHARD_OBSERVERCOUNT`; do

      runCommandInTerminal "$(assembleCommand_startObserverNode $SHARD $OBSERVER_INDEX)" $1

      let OBSERVER_INDEX++
    done
  done

  # Start Metachain Observers
  SHARD="metachain"
  for META_OBSERVER in `seq $META_OBSERVERCOUNT`; do

    runCommandInTerminal "$(assembleCommand_startObserverNode $SHARD $OBSERVER_INDEX)" $1

    let OBSERVER_INDEX++
  done
}

stopObservers() {
  OBSERVER_INDEX=0
  # Stop Shard Observers
  let "max_shard_id=$SHARDCOUNT - 1"
  for SHARD in `seq 0 1 $max_shard_id`; do
    for OBSERVER_IN_SHARD in `seq $SHARD_OBSERVERCOUNT`; do

      let "PORT = $PORT_ORIGIN_OBSERVER + $OBSERVER_INDEX"
      stopProcessByPort $PORT

      let OBSERVER_INDEX++
    done
  done

  # Start Metachain Observers
  SHARD="metachain"
  for META_OBSERVER in `seq $META_OBSERVERCOUNT`; do

      let "PORT = $PORT_ORIGIN_OBSERVER + $OBSERVER_INDEX"
      stopProcessByPort $PORT

    let OBSERVER_INDEX++
  done
}

startValidators() {
  setTerminalSession "elrond-nodes"
  setTerminalLayout "tiled"

  setWorkdirForNextCommands "$TESTNETDIR/node"

  # Start Shard Validators
  VALIDATOR_INDEX=0
  let "max_shard_id=$SHARDCOUNT - 1"
  for SHARD in `seq 0 1 $max_shard_id`; do
    for VALIDATOR_IN_SHARD in `seq $SHARD_VALIDATORCOUNT`; do

      runCommandInTerminal "$(assembleCommand_startValidatorNode $VALIDATOR_INDEX)" $1

      let VALIDATOR_INDEX++
    done
  done

  setTerminalSession "elrond-nodes"
  setTerminalLayout "tiled"
  # Start Metachain Validators
  SHARD="metachain"
  for META_VALIDATOR in `seq $META_VALIDATORCOUNT`; do

    runCommandInTerminal "$(assembleCommand_startValidatorNode $VALIDATOR_INDEX)" $1

    let VALIDATOR_INDEX++
  done
}

stopValidators() {
  VALIDATOR_INDEX=0
  let "max_shard_id=$SHARDCOUNT - 1"
  for SHARD in `seq 0 1 $max_shard_id`; do
    for VALIDATOR_IN_SHARD in `seq $SHARD_VALIDATORCOUNT`; do

      let "PORT = $PORT_ORIGIN_VALIDATOR + $VALIDATOR_INDEX"
      stopProcessByPort $PORT

      let VALIDATOR_INDEX++
    done
  done
}

assembleCommand_startObserverNode() {
  SHARD=$1
  OBSERVER_INDEX=$2
  let "PORT = $PORT_ORIGIN_OBSERVER + $OBSERVER_INDEX"
  let "RESTAPIPORT=$PORT_ORIGIN_OBSERVER_REST + $OBSERVER_INDEX"
  let "KEY_INDEX=$TOTAL_NODECOUNT - $OBSERVER_INDEX - 1"
  WORKING_DIR=$TESTNETDIR/node_working_dirs/observer$OBSERVER_INDEX

  local nodeCommand="./node \
        -port $PORT -log-save -log-level $LOGLEVEL -rest-api-interface localhost:$RESTAPIPORT \
        -destination-shard-as-observer $SHARD \
        -sk-index $KEY_INDEX \
        -working-directory $WORKING_DIR"

  if [ -n "$NODE_NICENESS" ]
  then
    nodeCommand="nice -n $NODE_NICENESS $nodeCommand"
  fi

  if [ $NODETERMUI -eq 0 ]
  then
    nodeCommand="$nodeCommand -use-log-view"
  fi

  echo $nodeCommand
}

assembleCommand_startValidatorNode() {
  VALIDATOR_INDEX=$1
  let "PORT = $PORT_ORIGIN_VALIDATOR + $VALIDATOR_INDEX"
  let "RESTAPIPORT=$PORT_ORIGIN_VALIDATOR_REST + $VALIDATOR_INDEX"
  let "KEY_INDEX=$VALIDATOR_INDEX"
  WORKING_DIR=$TESTNETDIR/node_working_dirs/validator$VALIDATOR_INDEX

  local nodeCommand="./node \
        -port $PORT -log-save -log-level $LOGLEVEL -rest-api-interface localhost:$RESTAPIPORT \
        -sk-index $KEY_INDEX \
        -working-directory $WORKING_DIR"

  if [ -n "$NODE_NICENESS" ]
  then
    nodeCommand="nice -n $NODE_NICENESS $nodeCommand"
  fi

  if [ $NODETERMUI -eq 0 ]
  then
    nodeCommand="$nodeCommand -use-log-view"
  fi

  echo $nodeCommand
}

