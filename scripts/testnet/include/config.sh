source "$ELRONDTESTNETSCRIPTSDIR/variables.sh"

generateConfig() {
  echo "Generating configuration using values from scripts/variables.sh..."

  pushd $TESTNETDIR/filegen
  ./filegen \
    -mint-value $MINT_VALUE                               \
    -num-of-shards $SHARDCOUNT                            \
    -num-of-nodes-in-each-shard $SHARD_VALIDATORCOUNT     \
    -num-of-observers-in-each-shard $SHARD_OBSERVERCOUNT  \
    -consensus-group-size $SHARD_CONSENSUS_SIZE           \
    -num-of-metachain-nodes $META_VALIDATORCOUNT          \
    -num-of-observers-in-metachain $META_OBSERVERCOUNT    \
    -metachain-consensus-group-size $META_CONSENSUS_SIZE  \
    -consensus-type $CONSENSUS_TYPE
  popd
}

copyConfig() {
  pushd $TESTNETDIR

  cp ./filegen/genesis.json ./node/config
  cp ./filegen/nodesSetup.json ./node/config
  cp ./filegen/initialBalancesPkPlain.txt ./node/config
  cp ./filegen/initialBalancesSk.pem ./node/config
  cp ./filegen/initialBalancesSkPlain.txt ./node/config
  cp ./filegen/initialNodesPkPlain.txt ./node/config
  cp ./filegen/initialNodesSk.pem ./node/config
  cp ./filegen/initialNodesSkPlain.txt ./node/config
  echo "Configuration files copied from the configuration generator to the working directories of the executables."
  popd
}

copySeednodeConfig() {
  pushd $TESTNETDIR
  cp $SEEDNODEDIR/config/p2p.toml ./seednode/config
  popd
}

updateSeednodeConfig() {
  pushd $TESTNETDIR/seednode/config
  cp p2p.toml p2p_edit.toml

  updateTOMLValue p2p_edit.toml "Port" $PORT_SEEDNODE

  cp p2p_edit.toml p2p.toml
  rm p2p_edit.toml

  echo "Updated configuration for the Seednode."
  popd
}

copyNodeConfig() {
  pushd $TESTNETDIR
  cp $NODEDIR/config/config.toml ./node/config
  cp $NODEDIR/config/economics.toml ./node/config
  cp $NODEDIR/config/prefs.toml ./node/config
  cp $NODEDIR/config/external.toml ./node/config
  cp $NODEDIR/config/p2p.toml ./node/config
  cp $NODEDIR/config/gasSchedule.toml ./node/config


  echo "Configuration files copied from the Node to the working directories of the executables."
  popd
}

updateNodeConfig() {
  pushd $TESTNETDIR/node/config
  cp p2p.toml p2p_edit.toml

  updateTOMLValue p2p_edit.toml "InitialPeerList" "[\"$P2P_SEEDNODE_ADDRESS\"]"

  cp p2p_edit.toml p2p.toml
  rm p2p_edit.toml

  cp nodesSetup.json nodesSetup_edit.json
  
  let startTime="$(date +%s) + $NODE_DELAY"
  updateJSONValue nodesSetup_edit.json "startTime" "$startTime"

	if [ $ALWAYS_NEW_CHAINID -eq 1 ]; then
		updateJSONValue nodesSetup_edit.json "chainID" "\"$startTime\""
	fi

  cp nodesSetup_edit.json nodesSetup.json
  rm nodesSetup_edit.json

  echo "Updated configuration for Nodes."
  popd
}

copyProxyConfig() {
  pushd $TESTNETDIR

  cp $PROXYDIR/config/config.toml ./proxy/config/

  cp ./node/config/economics.toml ./proxy/config/
  cp ./node/config/initialBalancesSk.pem ./proxy/config

  echo "Copied configuration for the Proxy."
  popd
}

updateProxyConfig() {
  pushd $TESTNETDIR/proxy/config
  cp config.toml config_edit.toml

  # Truncate config.toml before the [[Observers]] list
  sed -i -n '/\[\[Observers\]\]/q;p' config_edit.toml
  
  updateTOMLValue config_edit.toml "ServerPort" $PORT_PROXY
  generateProxyObserverList config_edit.toml

  cp config_edit.toml config.toml
  rm config_edit.toml

  echo "Updated configuration for the Proxy."
  popd
}

copyTxGenConfig() {
  pushd $TESTNETDIR

  cp $TXGENDIR/config/config.toml ./txgen/config/

  cp $TXGENDIR/config/sc.toml ./txgen/config/
  cp $TXGENDIR/config/*.wasm ./txgen/config/

  cp ./node/config/economics.toml ./txgen/config/
  cp ./node/config/initialBalancesSk.pem ./txgen/config

  echo "Copied configuration for the TxGen."
  popd
}

updateTxGenConfig() {
  pushd $TESTNETDIR/txgen/config
  cp config.toml config_edit.toml

  updateTOMLValue config_edit.toml "ServerPort" $PORT_TXGEN
  updateTOMLValue config_edit.toml "ProxyServerURL" "\"http://127.0.0.1:$PORT_PROXY\""

  cp config_edit.toml config.toml
  rm config_edit.toml

  echo "Updated configuration for the TxGen."
  popd
}


generateProxyObserverList() {
  OBSERVER_INDEX=0
  OUTPUTFILE=$!
  # Start Shard Observers
  let "max_shard_id=$SHARDCOUNT - 1"
  for SHARD in `seq 0 1 $max_shard_id`; do
    for OBSERVER_IN_SHARD in `seq $SHARD_OBSERVERCOUNT`; do
      let "PORT = $PORT_ORIGIN_OBSERVER_REST + $OBSERVER_INDEX"

      echo -n "[[Observers]]" >> config_edit.toml
      echo -n "   ShardId = $SHARD" >> config_edit.toml
      echo -n "   Address = \"http://127.0.0.1:$PORT\"" >> config_edit.toml
      echo -n ""$'\n' >> config_edit.toml

      let OBSERVER_INDEX++
    done
  done
}

updateTOMLValue() {
  local filename=$1
  local key=$2
  local value=$3

  escaped_value=$(printf "%q" $value)

  sed -i "s,$key = .*\$,$key = $escaped_value," $filename
}


updateJSONValue() {
  local filename=$1
  local key=$2
  local value=$3

  escaped_value=$(printf "%q" $value)

  sed -i "s,\"$key\": .*\$,\"$key\": $escaped_value\,," $filename
}
