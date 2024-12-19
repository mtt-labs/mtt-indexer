package cornjob

import (
	"context"
	sdkmath "cosmossdk.io/math"
	"github.com/DefiantLabs/probe/client"
	"github.com/syndtr/goleveldb/leveldb"
	"mtt-indexer/db"
	"mtt-indexer/logger"
	"mtt-indexer/rpc"
	"mtt-indexer/types"
	"mtt-indexer/util/cron"
	"time"
)

type TotalStakeJob struct {
	ldb *db.LDB
	cl  *client.ChainClient
}

func CronJobLedgerInit(db *db.LDB, cl *client.ChainClient) {
	c := cron.NewCron()
	//0 0 */8 * * *
	//0 0 0 * * *
	c.Register("Ledger job", "0 0 0 * * *", NewTotalStakeJob(db, cl).saveValidatorsReward)
	c.Run()
	defer c.Stop()
}

func NewTotalStakeJob(ldb *db.LDB, cl *client.ChainClient) *TotalStakeJob {
	return &TotalStakeJob{ldb: ldb, cl: cl}
}

func (t *TotalStakeJob) saveValidatorsReward(ctx context.Context) error {
	validators, err := rpc.AllValidator(t.cl)
	if err != nil {
		return err
	}
	for _, v := range validators {
		err = t.saveValidatorReward(v)
		if err != nil {
			logger.Logger.Errorf("t.saveValidatorReward %s ,err %v", v, err)
			return err
		}
	}
	return err
}

func (t *TotalStakeJob) saveValidatorReward(validator string) error {
	reward, err := rpc.GetValidatorReward(t.cl, validator)
	if err != nil {
		return nil
	}
	record := &types.RewardRecord{
		Validator: validator,
		Amount:    reward.String(),
		Time:      time.Now().Unix(),
	}
	recordAmount, _ := sdkmath.NewIntFromString(record.Amount)
	claimed, err := t.getValidatorClaimed24H(validator)
	if err != nil {
		return err
	}
	record.Amount = recordAmount.Sub(claimed).String()
	return t.ldb.Transaction(
		func(l *db.LDB, batch *leveldb.Batch) error {
			err := db.StoreRecord(l.DB, batch, record)
			if err != nil {
				return err
			}
			return nil
		})
}

func (t *TotalStakeJob) getValidatorClaimed24H(validator string) (sdkmath.Int, error) {
	IRecord, err := t.ldb.GetRecordByType(&types.Claimed24H{Validator: validator})
	if err != nil {
		return sdkmath.NewInt(0), err
	}
	storeRecord := &types.Claimed24H{}
	if IRecord == nil {
		return sdkmath.NewInt(0), nil
	} else {
		if record, ok := IRecord.(*types.Claimed24H); ok {
			storeRecord = record
		}
		amount, _ := sdkmath.NewIntFromString(storeRecord.Amount)
		return amount, nil
	}
}
