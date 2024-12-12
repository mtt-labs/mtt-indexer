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
type MsgDelegateUndelegateParser struct {
	Id string
}

func (c *MsgDelegateUndelegateParser) Identifier() string {
	return c.Id
}

func (c *MsgDelegateUndelegateParser) ParseMessage(cosmosMsg stdTypes.Msg, log *indexerTxTypes.LogMessage) (*any, error) {
	msgDelegate, okMsgDelegate := cosmosMsg.(*stakingTypes.MsgDelegate)
	msgUndelegate, okMsgUndelegate := cosmosMsg.(*stakingTypes.MsgUndelegate)
	if !okMsgDelegate && !okMsgUndelegate {
		return nil, errors.New("not a delegation message")
	}

	if okMsgDelegate {
		delegator := types.Address{
			Address: msgDelegate.DelegatorAddress,
		}

		validator := Validator{
			ValidatorAddress: types.Address{
				Address: msgDelegate.ValidatorAddress,
			},
		}

		amount := msgDelegate.Amount.Amount.String()
		denom := types.Denom{
			Base: msgDelegate.Amount.Denom,
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

	delegator := types.Address{
		Address: msgUndelegate.DelegatorAddress,
	}

	validator := Validator{
		ValidatorAddress: types.Address{
			Address: msgUndelegate.ValidatorAddress,
		},
	}

	amount := msgUndelegate.Amount.Amount.String()
	denom := types.Denom{
		Base: msgUndelegate.Amount.Denom,
	}

	storageVal := any(types.ValidatorRecord{
		Delegator:      delegator.Address,
		Validator:      validator.ValidatorAddress.Address,
		Amount:         amount,
		Denom:          denom.Base,
		DelegationType: types.Undelegate,
	})

	return &storageVal, nil
}

func (c *MsgDelegateUndelegateParser) IndexMessage(ldb *db.LDB, batch *leveldb.Batch, txhash string, dataset *any, message types.Message, messageEvents []MessageEventWithAttributes) error {
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
	if validatorRecord.DelegationType == types.Delegate {
		outList.AddValidatorRecord(validatorRecord, true)
	} else {
		outList.AddValidatorRecord(validatorRecord, false)
	}

	err = db.StoreRecord(ldb.DB, batch, outList)
	if err != nil {
		return err
	}

	return db.StoreRecord(ldb.DB, batch, validatorRecord.ToDelegate())
}

type MsgUndelegateParser struct{}

type Validator struct {
	ID                 uint
	ValidatorAddress   types.Address
	ValidatorAddressID uint
}
