package main

const NETWORK_ADRECORD = 1
const NETWORK_TRADEDOUBLER = 2

type networkinterface interface {
	parseProducts(f *feed) ([]product, error)
}
