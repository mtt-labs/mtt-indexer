package db

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"mtt-indexer/types"
	"testing"
	"time"
)

var (
	tailFix = "main"
)

func TestDb(t *testing.T) {
	db := NewLdb(tailFix)

	delegator := ""
	recordsIFace, total, err := db.GetAllRecordsWithAutoId(&types.DelegatorRecord{}, 10, 0, false)
	if err != nil {
		t.Error(err)
	} else {
		for _, record := range recordsIFace {
			if delegatorRecord, ok := record.(*types.DelegatorRecord); ok {
				fmt.Printf("Delegator Record: %+v\n", delegatorRecord)
				delegator = delegatorRecord.Delegator
			}
		}
	}
	fmt.Printf("total: %d\n", total)

	recordsIFace, total, err = db.GetAllRecordsWithAutoId(&types.ValidatorRecord{}, 10, 0, false)
	if err != nil {
		t.Error(err)
	} else {
		for _, record := range recordsIFace {
			if delegatorRecord, ok := record.(*types.ValidatorRecord); ok {
				fmt.Printf("ValidatorRecord Record: %+v\n", delegatorRecord)
			}
		}
	}
	fmt.Printf("total: %d\n", total)

	outList := &types.DelegatorOutList{
		Delegator: delegator,
	}
	record, err := db.GetRecordByType(outList)
	if err != nil {
		t.Error(err)
	}
	if record != nil {
		if delegatorOutList, ok := record.(*types.DelegatorOutList); ok {
			outList = delegatorOutList
		}
	}

	chain := &types.Chain{
		Name: "mtt",
	}
	recordInterface, err := db.GetRecordByType(chain)
	if err != nil {
		return
	}
	if record != nil {
		if chainRecord, ok := recordInterface.(*types.Chain); ok {
			chain = chainRecord
		}
	}
	fmt.Printf("outList: %+v\n", chain)
}

func TestDbH(t *testing.T) {
	db := NewLdb(tailFix)

	time, _ := time.Parse(time.RFC3339, "2024-06-11T10:51:01.477179159Z")
	vRecord := &types.ValidatorRecord{
		Delegator:      "mtt10wpwl4mqpgdgz8597kphgahx3a8degvg58kjx5",
		Validator:      "mttvaloper10wpwl4mqpgdgz8597kphgahx3a8degvgtnmw9f",
		Amount:         "1000000000000000000000000",
		Denom:          "amtt",
		TxHash:         "GENESIS TX",
		DelegationType: types.Delegate,
		DelegationTime: time,
	}
	err := db.Transaction(func(db *LDB, batch *leveldb.Batch) error {
		err := StoreRecord(db.DB, batch, vRecord)
		if err != nil {
			return err
		}
		err = StoreRecord(db.DB, batch, vRecord.ToDelegate())
		if err != nil {
			return err
		}
		outList := &types.DelegatorOutList{
			Delegator:  vRecord.Delegator,
			Validators: []string{},
			Amounts:    []string{},
			Denom:      vRecord.Denom,
		}

		outList.AddValidatorRecord(*vRecord, true)
		err = StoreRecord(db.DB, batch, outList)
		if err != nil {
			return err
		}

		commissionRecord := &types.CommissionRecord{
			Validator:  vRecord.Validator,
			Commission: 0.1,
			Time:       time,
		}
		err = StoreRecord(db.DB, batch, commissionRecord)
		if err != nil {
			return err
		}

		chain := &types.Chain{
			Name:    "mtt",
			Rpc:     "https://cosmos-rpc.mtt.network:443",
			ChainID: "mtt_6880-1",
			Height:  5054844,
			//Height: 5211170,
		}
		err = StoreRecord(db.DB, batch, chain)
		if err != nil {
			return err
		}

		stake := &types.Stake{
			Amount: "1000000000000000000000000",
		}
		err = StoreRecord(db.DB, batch, stake)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return
	}

	//2024-06-11T10:51:01.477179159Z
}
