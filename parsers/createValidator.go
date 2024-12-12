package parsers

import (
	"errors"
	stdTypes "github.com/cosmos/cosmos-sdk/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/syndtr/goleveldb/leveldb"
	indexerTxTypes "mtt-indexer/cosmos/modules/tx"
	"mtt-indexer/db"
	"mtt-indexer/types"
	"time"
)

// This defines the custom message parsers for the delegation and undelegation message type
// It implements the MessageParser interface
type MsgCreateValidatorParser struct {
	Id string
}

func (c *MsgCreateValidatorParser) Identifier() string {
	return c.Id
}

func (c *MsgCreateValidatorParser) ParseMessage(cosmosMsg stdTypes.Msg, log *indexerTxTypes.LogMessage) (*any, error) {
	msgCreateValidator, ok := cosmosMsg.(*stakingTypes.MsgCreateValidator)
	if !ok {
		return nil, errors.New("not a delegation message")
	}

	delegator := types.Address{
		Address: msgCreateValidator.DelegatorAddress,
	}

	validator := Validator{
		ValidatorAddress: types.Address{
			Address: msgCreateValidator.ValidatorAddress,
		},
	}

	amount := msgCreateValidator.Value.Amount.String()
	denom := types.Denom{
		Base: msgCreateValidator.Value.Denom,
	}

	storageVal := any(types.ValidatorRecord{
		Delegator:      delegator.Address,
		Validator:      validator.ValidatorAddress.Address,
		Amount:         amount,
		Denom:          denom.Base,
		DelegationType: types.Delegate,
	})
	return &storageVal, nil
}

func (c *MsgCreateValidatorParser) IndexMessage(ldb *db.LDB, batch *leveldb.Batch, txhash string, dataset *any, message types.Message, messageEvents []MessageEventWithAttributes) error {
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

	return db.StoreRecord(ldb.DB, batch, validatorRecord.ToDelegate())
}

type CommissionChangeEvent struct {
	ID         uint
	Validator  Validator
	Commission float64
	Time       time.Time
}
