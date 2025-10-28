package network

// Network ports
const (
	MAINNET_PORT int = 8333
	TESTNET_PORT int = 18333
)

// DNS seeds for peer discovery
const (
	MAINNET_SEEDS string = "seed.bitcoin.sipa.be"
	TESTNET_SEEDS string = "testnet-seed.bitcoin.jonasschnelli.ch"
)

// Service flags (NODE_* constants)
const (
	NODE_NETWORK         uint64 = 1 << 0 // NODE_NETWORK (bit 0) - Full node
	NODE_GETUTXO         uint64 = 1 << 1 // NODE_GETUTXO (bit 1) - BIP 64
	NODE_BLOOM           uint64 = 1 << 2 // NODE_BLOOM (bit 2) - BIP 37
	NODE_WITNESS         uint64 = 1 << 3 // NODE_WITNESS (bit 3) - BIP 144
	NODE_XTHIN           uint64 = 1 << 4 // NODE_XTHIN (bit 4)
	NODE_COMPACT_FILTERS uint64 = 1 << 6 // NODE_COMPACT_FILTERS (bit 6) - BIP 157
	NODE_NETWORK_LIMITED uint64 = 1 << 10 // NODE_NETWORK_LIMITED (bit 10) - BIP 159
)
