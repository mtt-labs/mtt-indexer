package filter

import (
	"github.com/cosmos/cosmos-sdk/types"
	"mtt-indexer/cosmos/modules/tx"
)

type MessageFilter interface {
	ShouldIndex(types.Msg, tx.LogMessage) bool
}
