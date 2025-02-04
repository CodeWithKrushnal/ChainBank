package ethereum

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

var EthereumClient *ethclient.Client

func InitEthereumClient(rpcURL string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	EthereumClient = client

	log.Printf("Ethereum Client Started on: %v", rpcURL)
	return EthereumClient, nil
}
