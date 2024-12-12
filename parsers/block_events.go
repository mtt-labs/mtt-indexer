package parsers

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"mtt-indexer/types"
)

type BlockEventParser interface {
	Identifier() string
	ParseBlockEvent(abci.Event) (*any, error)
	IndexBlockEvent(*any, types.Block, types.BlockEvent, []types.BlockEventAttribute) error
}

type BlockEventParsedData struct {
	Data   *any
	Error  error
	Parser *BlockEventParser
}
