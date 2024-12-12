package types

import "fmt"

type Chain struct {
	Name    string
	Rpc     string
	ChainID string
	Height  int64
}

func (c *Chain) Key() string {
	return fmt.Sprintf("Chain_%s", c.Name)
}

func (c *Chain) Clone() *Chain {
	return &Chain{
		Name:    c.Name,
		Rpc:     c.Rpc,
		ChainID: c.ChainID,
		Height:  c.Height,
	}
}
