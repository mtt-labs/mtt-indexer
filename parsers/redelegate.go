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
type MsgRedelegateParser struct {
	Id string
}

func (c *MsgRedelegateParser) Identifier() string {
	return c.Id
}

func (c *MsgRedelegateParser) ParseMessage(cosmosMsg stdTypes.Msg, log *indexerTxTypes.LogMessage) (*any, error) {
	msg, ok := cosmosMsg.(*stakingTypes.MsgBeginRedelegate)

	if !ok {
		return nil, errors.New("not a delegation message")
	}

	storageVal := any(types.RedelegateRecord{
		Delegator:      msg.DelegatorAddress,
		Src:            msg.ValidatorSrcAddress,
		Dst:            msg.ValidatorDstAddress,
		Amount:         msg.Amount.Amount.String(),
		Denom:          msg.Amount.Denom,
		DelegationType: types.Redelegate,
	})

	return &storageVal, nil

}

func (c *MsgRedelegateParser) IndexMessage(ldb *db.LDB, batch *leveldb.Batch, txhash string, dataset *any, message types.Message, messageEvents []MessageEventWithAttributes) error {
	//src
	record, ok := (*dataset).(types.RedelegateRecord)
	if !ok {
		return errors.New("not a ValidatorRecord type")
	}

	validatorSrcRecord := &types.ValidatorRecord{
		Delegator:      record.Delegator,
		Validator:      record.Src,
		Amount:         record.Amount,
		Denom:          record.Denom,
		TxHash:         txhash,
		DelegationType: types.Redelegate,
		DelegationTime: message.Tx.Block.TimeStamp,
	}

	err := db.StoreRecord(ldb.DB, batch, validatorSrcRecord)
	if err != nil {
		return err
	}

	outList := &types.DelegatorOutList{
		Delegator: validatorSrcRecord.Delegator,
	}
	recordOutList, err := ldb.GetRecordByType(outList)
	if err != nil {
		return err
	}
	if recordOutList != nil {
		if delegatorOutList, ok := recordOutList.(*types.DelegatorOutList); ok {
			outList = delegatorOutList
		}
	} else {
		outList = &types.DelegatorOutList{
			Delegator:  validatorSrcRecord.Delegator,
			Validators: []string{},
			Amounts:    []string{},
			Denom:      validatorSrcRecord.Denom,
		}
	}
	outList.AddValidatorRecord(*validatorSrcRecord, false)
	err = db.StoreRecord(ldb.DB, batch, outList)
	if err != nil {
		return err
	}

	err = db.StoreRecord(ldb.DB, batch, validatorSrcRecord.ToDelegate())
	if err != nil {
		return err
	}

	//dst
	validatorSrcRecord.Validator = record.Dst
	err = db.StoreRecord(ldb.DB, batch, validatorSrcRecord)
	if err != nil {
		return err
	}
	outList.AddValidatorRecord(*validatorSrcRecord, true)
	err = db.StoreRecord(ldb.DB, batch, outList)
	if err != nil {
		return err
	}
	err = db.StoreRecord(ldb.DB, batch, validatorSrcRecord.ToDelegate())
	if err != nil {
		return err
	}
	return nil
}
