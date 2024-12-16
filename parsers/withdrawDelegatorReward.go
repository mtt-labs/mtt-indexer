package parsers

import (
	"errors"
	stdTypes "github.com/cosmos/cosmos-sdk/types"
	distributionTypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/syndtr/goleveldb/leveldb"
	indexerTxTypes "mtt-indexer/cosmos/modules/tx"
	"mtt-indexer/db"
	"mtt-indexer/types"
)

// This defines the custom message parsers for the delegation and undelegation message type
// It implements the MessageParser interface
type MsgWithdrawDelegatorRewardParser struct {
	Id string
}

func (c *MsgWithdrawDelegatorRewardParser) Identifier() string {
	return c.Id
}

func (c *MsgWithdrawDelegatorRewardParser) ParseMessage(cosmosMsg stdTypes.Msg, log *indexerTxTypes.LogMessage) (*any, error) {
	msgWithdrawDelegator, ok := cosmosMsg.(*distributionTypes.MsgWithdrawDelegatorReward)
	if !ok {
		return nil, errors.New("not a delegation message")
	}

	amount := ""
	if len(log.Events) > 0 {
		if log.Events[len(log.Events)-1].Attributes[0].Key == "amount" {
			amount = log.Events[len(log.Events)-1].Attributes[0].Value
			amount = amount[0 : len(amount)-4]
		}
	}

	storageVal := any(types.ValidatorRecord{
		Delegator:      msgWithdrawDelegator.DelegatorAddress,
		Validator:      msgWithdrawDelegator.ValidatorAddress,
		Amount:         amount,
		Denom:          "amtt",
		DelegationType: types.Claim,
	})

	return &storageVal, nil
}

func (c *MsgWithdrawDelegatorRewardParser) IndexMessage(ldb *db.LDB, batch *leveldb.Batch, txhash string, dataset *any, message types.Message, messageEvents []MessageEventWithAttributes) error {
	validatorRecord, ok := (*dataset).(types.ValidatorRecord)
	if !ok {
		return errors.New("not a ValidatorRecord type")
	}
	validatorRecord.TxHash = txhash
	validatorRecord.DelegationTime = message.Tx.Block.TimeStamp
	err := db.StoreRecord(ldb.DB, batch, &validatorRecord)
	if err != nil {
		return err
	}

	outList := &types.DelegatorOutList{
		Delegator: validatorRecord.Delegator,
	}
	record, err := ldb.GetRecordByType(outList)
	if err != nil {
		return err
	}
	if record != nil {
		if delegatorOutList, ok := record.(*types.DelegatorOutList); ok {
			outList = delegatorOutList
		}
	} else {
		outList = &types.DelegatorOutList{
			Delegator:  validatorRecord.Delegator,
			Validators: []string{},
			Amounts:    []string{},
			Denom:      validatorRecord.Denom,
		}
	}
	outList.AddValidatorRecord(validatorRecord, false)
	err = db.StoreRecord(ldb.DB, batch, outList)
	if err != nil {
		return err
	}

	return db.StoreRecord(ldb.DB, batch, validatorRecord.ToDelegate())
}
