package ethereum

import (
	"fmt"
	"log/slog"

	"github.com/CodeWithKrushnal/ChainBank/utils"
	"github.com/ethereum/go-ethereum/ethclient"
)

var EthereumClient *ethclient.Client

// InitEthereumClient initializes the Ethereum client using the provided RPC URL.
// It returns a pointer to the ethclient.Client instance and an error if any occurs.
func InitEthereumClient(rpcURL string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrEthereumClientInit, err)
	}
	EthereumClient = client

	slog.Info("Ethereum Client Started", "rpcURL", rpcURL)
	return EthereumClient, nil
}
