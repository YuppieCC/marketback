package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	log "github.com/sirupsen/logrus"

	mcsolana "marketcontrol/pkg/solana"
)

func main() {
	// Define command line flags
	mintAddr := flag.String("mint", "So11111111111111111111111111111111111111112", "Token mint address to fetch metadata for")
	rpcURL := flag.String("rpc-url", "https://red-wider-scion.solana-mainnet.quiknode.pro/7d63bea9a0a2d0a3664671d551a2d3565bef43b6/", "Solana RPC URL")
	flag.Parse()

	// Validate required flags
	if *mintAddr == "" {
		log.Error("Token mint address is required")
		fmt.Println("Usage example: go run scripts/get_token_metadata.go -mint EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
		os.Exit(1)
	}

	// Initialize RPC client
	client := rpc.New(*rpcURL)

	// Parse mint address
	mint := solana.MustPublicKeyFromBase58(*mintAddr)

	// Fetch token metadata
	metadata, err := mcsolana.GetTokenMetadata(client, mint)
	if err != nil {
		log.Fatalf("Failed to fetch token metadata: %v", err)
	}

	if metadata == nil {
		log.Info("No metadata found for the specified token")
		return
	}

	// Print metadata in a formatted way
	fmt.Printf("\nToken Metadata for %s:\n", *mintAddr)
	fmt.Printf("Name: %s\n", metadata.Name)
	fmt.Printf("Symbol: %s\n", metadata.Symbol)
	fmt.Printf("Uri: %s\n", metadata.Uri)
	// fmt.Printf("Image: %s\n", metadata.Image)
	// fmt.Printf("Decimals: %d\n", metadata.Decimals)
}
