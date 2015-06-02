package main

const NETWORK_ADRECORD = 1
const NETWORK_TRADEDOUBLER = 2
const NETWORK_ADTRACTION = 3

type networkinterface interface {
	parseProducts(f *feed) ([]product, error)
}
