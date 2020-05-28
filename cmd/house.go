package main

// House is a dwelling place at an address.
type House struct {
	ID         int32  `json:"id"`
	AddressOne string `json:"addressOne"`
	AddressTwo string `json:"addressTwo,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	Zip        string `json:"zip"`
}
