package parsers

import (
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/syndtr/goleveldb/leveldb"
	txtypes "mtt-indexer/cosmos/modules/tx"
	"mtt-indexer/db"
	"mtt-indexer/types"
)

// Intermediate type for the database inserted message datasets
// Is there a way to remove this? It may require a one-to-many mapping of the message events + attributes instead of the belongs-to
type MessageEventWithAttributes struct {
	Event      types.MessageEvent
	Attributes []types.MessageEventAttribute
}

type MessageParser interface {
	Identifier() string
	ParseMessage(sdkTypes.Msg, *txtypes.LogMessage) (*any, error)
	IndexMessage(*db.LDB, *leveldb.Batch, string, *any, types.Message, []MessageEventWithAttributes) error
}

type MessageParsedData struct {
	Data   *any
	Error  error
	Parser *MessageParser
}
