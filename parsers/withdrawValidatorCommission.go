package parsers

import (
	sdkmath "cosmossdk.io/math"
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
type MsgWithdrawValidatorCommission struct {
	Id string
}

func (c *MsgWithdrawValidatorCommission) Identifier() string {
	return c.Id
}

func (c *MsgWithdrawValidatorCommission) ParseMessage(cosmosMsg stdTypes.Msg, log *indexerTxTypes.LogMessage) (*any, error) {
	msgWithdrawDelegator, ok := cosmosMsg.(*distributionTypes.MsgWithdrawValidatorCommission)
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

	storageVal := any(types.RewardRecord{
		Validator: msgWithdrawDelegator.ValidatorAddress,
		Amount:    amount,
	})

	return &storageVal, nil
}

func (c *MsgWithdrawValidatorCommission) IndexMessage(ldb *db.LDB, batch *leveldb.Batch, txhash string, dataset *any, message types.Message, messageEvents []MessageEventWithAttributes) error {
	rewardRecord, ok := (*dataset).(types.RewardRecord)
	if !ok {
		return errors.New("not a RewardRecord type")
	}

	//save reward record

	IRecord, err := ldb.GetRecordByType(&types.Claimed24H{Validator: rewardRecord.Validator})
	if err != nil {
		return err
	}
	storeRecord := &types.Claimed24H{}
	if IRecord == nil {
		storeRecord = &types.Claimed24H{
			Validator: rewardRecord.Validator,
			Amount:    rewardRecord.Amount,
		}
	} else {
		if record, ok := IRecord.(*types.Claimed24H); ok {
			storeRecord = record
		}
		amount, _ := sdkmath.NewIntFromString(storeRecord.Amount)
		claimAmount, _ := sdkmath.NewIntFromString(rewardRecord.Amount)
		amount = amount.Add(claimAmount)
		storeRecord.Amount = amount.String()
	}
	return db.StoreRecord(ldb.DB, batch, storeRecord)
}
