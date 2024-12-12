package parsers

import (
	"errors"
	stdTypes "github.com/cosmos/cosmos-sdk/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/syndtr/goleveldb/leveldb"
	indexerTxTypes "mtt-indexer/cosmos/modules/tx"
	"mtt-indexer/db"
	"mtt-indexer/types"
)

// This defines the custom message parsers for the delegation and undelegation message type
// It implements the MessageParser interface
type MsgEditValidatorParser struct {
	Id string
}

func (c *MsgEditValidatorParser) Identifier() string {
	return c.Id
}

func (c *MsgEditValidatorParser) ParseMessage(cosmosMsg stdTypes.Msg, log *indexerTxTypes.LogMessage) (*any, error) {
	msgEditValidator, ok := cosmosMsg.(*stakingTypes.MsgEditValidator)
	if !ok {
		return nil, errors.New("not a delegation message")
	}

	validator := Validator{
		ValidatorAddress: types.Address{
			Address: msgEditValidator.ValidatorAddress,
		},
	}

	commission := msgEditValidator.CommissionRate.MustFloat64()

	storageVal := any(CommissionChangeEvent{
		Validator:  validator,
		Commission: commission,
	})

	return &storageVal, nil
}

func (c *MsgEditValidatorParser) IndexMessage(ldb *db.LDB, batch *leveldb.Batch, txhash string, dataset *any, message types.Message, messageEvents []MessageEventWithAttributes) error {
	commissionRecord, ok := (*dataset).(types.CommissionRecord)
	if !ok {
		return errors.New("not a delegation event type")
	}
	commissionRecord.Time = message.Tx.Block.TimeStamp
	return db.StoreRecord(ldb.DB, batch, &commissionRecord)
}
