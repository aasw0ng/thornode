package main

import (
	"log"
	"flag"

	"gitlab.com/thorchain/bepswap/statechain/x/smoke"
)

func main() {
	apiAddr := flag.String("a", "testnet-dex.binance.org", "Binance API Address.")
	masterKey := flag.String("m", "", "The master private key.")
	poolKey := flag.String("p", "", "The pool private key.")
	environment := flag.String("e", "stage", "The environment to use [stage|dev|prod].")
	config := flag.String("c", "", "Path to the config file.")
	network := flag.Int("n", 0, "The network to use.")
	debug := flag.Bool("d", false, "Enable debugging of the Binance transactions.")
	flag.Parse()

	if *masterKey == "" {
		log.Fatal("No master key set!")
	}

	if *poolKey == "" {
		log.Fatal("No pool key set!")
	}

	if *config == "" {
		log.Fatal("No config file provided!")
	}

	s := smoke.NewSmoke(*apiAddr, *masterKey, *poolKey, *environment, *config, *network, *debug)
	s.Run()
}
