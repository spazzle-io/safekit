package chain

import "math/big"

// Ethereum

var Ethereum = &Chain{
	ID:   big.NewInt(1),
	Name: "Ethereum Mainnet",
	IsL2: false,
}

var Sepolia = &Chain{
	ID:   big.NewInt(11155111),
	Name: "Sepolia Testnet",
	IsL2: false,
}

// Polygon

var Polygon = &Chain{
	ID:   big.NewInt(137),
	Name: "Polygon POS",
	IsL2: true,
}

var PolygonZkEVM = &Chain{
	ID:   big.NewInt(1101),
	Name: "Polygon zkEVM",
	IsL2: true,
}

var PolygonAmoy = &Chain{
	ID:   big.NewInt(80002),
	Name: "Polygon Amoy Testnet",
	IsL2: true,
}

// Arbitrum

var ArbitrumOne = &Chain{
	ID:   big.NewInt(42161),
	Name: "Arbitrum One",
	IsL2: true,
}

var ArbitrumNova = &Chain{
	ID:   big.NewInt(42170),
	Name: "Arbitrum Nova",
	IsL2: true,
}

var ArbitrumSepolia = &Chain{
	ID:   big.NewInt(421614),
	Name: "Arbitrum Sepolia Testnet",
	IsL2: true,
}

// Base

var Base = &Chain{
	ID:   big.NewInt(8453),
	Name: "Base",
	IsL2: true,
}

var BaseSepolia = &Chain{
	ID:   big.NewInt(84532),
	Name: "Base Sepolia Testnet",
	IsL2: true,
}

// Optimism

var Optimism = &Chain{
	ID:   big.NewInt(10),
	Name: "Optimism",
	IsL2: true,
}

var OptimismSepolia = &Chain{
	ID:   big.NewInt(11155420),
	Name: "Optimism Sepolia Testnet",
	IsL2: true,
}

// BNB Smart Chain

var BNBSmartChain = &Chain{
	ID:   big.NewInt(56),
	Name: "BNB Smart Chain",
	IsL2: false,
}

var BNBSmartChainTestnet = &Chain{
	ID:   big.NewInt(97),
	Name: "BNB Smart Chain Testnet",
	IsL2: false,
}
