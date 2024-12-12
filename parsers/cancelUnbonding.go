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
type MsgCancelUnbondingParser struct {
	Id string
}

func (c *MsgCancelUnbondingParser) Identifier() string {
	return c.Id
}

func (c *MsgCancelUnbondingParser) ParseMessage(cosmosMsg stdTypes.Msg, log *indexerTxTypes.LogMessage) (*any, error) {
	msg, ok := cosmosMsg.(*stakingTypes.MsgCancelUnbondingDelegation)

	if !ok {
		return nil, errors.New("not a delegation message")
	}

	delegator := types.Address{
		Address: msg.DelegatorAddress,
	}

	validator := Validator{
		ValidatorAddress: types.Address{
			Address: msg.ValidatorAddress,
		},
	}

	amount := msg.Amount.Amount.String()
	denom := types.Denom{
		Base: msg.Amount.Denom,
	}

	storageVal := any(types.ValidatorRecord{
		Delegator:      delegator.Address,
		Validator:      validator.ValidatorAddress.Address,
		Amount:         amount,
		Denom:          denom.Base,
		DelegationType: types.CancelUnbonding,
	})

	return &storageVal, nil

}

func (c *MsgCancelUnbondingParser) IndexMessage(ldb *db.LDB, batch *leveldb.Batch, txhash string, dataset *any, message types.Message, messageEvents []MessageEventWithAttributes) error {
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
	outList.AddValidatorRecord(validatorRecord, true)
	err = db.StoreRecord(ldb.DB, batch, outList)
	if err != nil {
		return err
	}

	return db.StoreRecord(ldb.DB, batch, validatorRecord.ToDelegate())
}
