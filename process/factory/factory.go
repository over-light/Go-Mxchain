package factory

const (
	// TransactionTopic is the topic used for sharing transactions
	TransactionTopic = "transactions"
	// UnsignedTransactionTopic is the topic used for sharing unsigned transactions
	UnsignedTransactionTopic = "unsignedTransactions"
	// HeadersTopic is the topic used for sharing block headers
	HeadersTopic = "headers"
	// MiniBlocksTopic is the topic used for sharing mini blocks
	MiniBlocksTopic = "txBlockBodies"
	// PeerChBodyTopic is used for sharing peer change block bodies
	PeerChBodyTopic = "peerChangeBlockBodies"
	// MetachainBlocksTopic is used for sharing metachain block headers between shards
	MetachainBlocksTopic = "metachainBlocks"
	// ShardHeadersForMetachainTopic is used for sharing shards block headers to the metachain nodes
	ShardHeadersForMetachainTopic = "shardHeadersForMetachain"
)

// InternalTestingVM is a byte array identified for the smart contract address created for the testing VM
var InternalTestingVM = []byte{0, 0}

// IELEVirtualMachine is a byte array identifier for the smart contract address created for IELE VM
var IELEVirtualMachine = []byte{1, 0}

// HeraWABTVirtualMachine is a byte array identifier for the smart contract address created for Hera WABT VM
var HeraWABTVirtualMachine = []byte{2, 0}

// HeraWAVMVirtualMachine is a byte array identifier for the smart contract address created for Hera WAVM VM
var HeraWAVMVirtualMachine = []byte{2, 0}

// HeraWBinaryenVirtualMachine is a byte array identifier for the smart contract address created for Hera Binaryen VM
var HeraWBinaryenVirtualMachine = []byte{2, 0}
