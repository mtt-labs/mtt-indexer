package cornjob

import (
	"context"
	"github.com/syndtr/goleveldb/leveldb"
	"mtt-indexer/db"
	"mtt-indexer/logger"
	"mtt-indexer/types"
	"mtt-indexer/util/cron"
	"time"
)

type TotalStakeJob struct {
	ldb *db.LDB
}

func CronJobLedgerInit(db *db.LDB) {
	c := cron.NewCron()
	//0 0 */8 * * *
	//0 0 0 * * *
	c.Register("Ledger job", "0 0 0 * * *", NewTotalStakeJob(db).saveStake)
	c.Run()
	defer c.Stop()
}

func NewTotalStakeJob(ldb *db.LDB) *TotalStakeJob {
	return &TotalStakeJob{ldb: ldb}
}

func (t *TotalStakeJob) saveStake(ctx context.Context) error {
	stake := t.getCulStake()

	stakeHis := &types.StakeHistory{
		ID:     0,
		Amount: stake.Amount,
		Time:   time.Now(),
	}
	err := t.ldb.Transaction(
		func(l *db.LDB, batch *leveldb.Batch) error {
			err := db.StoreRecord(l.DB, batch, stakeHis)
			if err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		logger.Logger.Errorf("corn job t.ldb.Transaction : %v", err)
	}
	return err
}

func (t *TotalStakeJob) getCulStake() *types.Stake {
	stake := &types.Stake{}
	record, err := t.ldb.GetRecordByType(stake)
	if err != nil {
		return nil
	}
	if record != nil {
		if stakeRecord, ok := record.(*types.Stake); ok {
			stake = stakeRecord
		}
	}
	return stake
}
