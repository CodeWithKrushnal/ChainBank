package ethereum

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

var EthereumClient *ethclient.Client

func InitEthereumClient(rpcURL string) error {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return err
	}
	EthereumClient = client

	log.Printf("Ethereum Client Started on: %v", rpcURL)
	return nil
}
