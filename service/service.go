package service

import (
	"mtt-indexer/db"
	"mtt-indexer/types"
	"time"
)

type IService interface {
	GetChainHeight() (int64, error)
	GetDelegatorList(delegator string) (*types.DelegatorOutList, error)
	GetDelegatorHistory(delegator string, limit, offset int, asc bool) ([]*types.DelegatorRecord, int, error)
	GetValidatorHistory(Validator string, limit, offset int, asc bool) ([]*types.ValidatorRecord, int, error)
	GetCommissionRecord(Validator string, limit, offset int) ([]*types.CommissionRecord, int, error)
	GetStakeHistory(limit, offset int) ([]*types.StakeHistory, int, error)
}

type Service struct {
	ldb *db.LDB
}

func NewService(db *db.LDB) *Service {
	return &Service{ldb: db}
}

func (s *Service) GetChainHeight() (int64, error) {
	chain := &types.Chain{
		Name: "mtt",
	}
	record, err := s.ldb.GetRecordByType(chain)
	if err != nil {
		return 0, err
	}
	if record != nil {
		if chainRecord, ok := record.(*types.Chain); ok {
			chain = chainRecord
		}
	}
	return chain.Height, nil
}

func (s *Service) GetDelegatorList(delegator string) (*types.DelegatorOutList, error) {
	outList := &types.DelegatorOutList{
		Delegator: delegator,
	}
	record, err := s.ldb.GetRecordByType(outList)
	if err != nil {
		return nil, err
	}
	if record != nil {
		if delegatorOutList, ok := record.(*types.DelegatorOutList); ok {
			outList = delegatorOutList
		}
	}
	return outList, nil
}

func (s *Service) GetDelegatorHistory(delegator string, limit, offset int, asc bool) ([]*types.DelegatorRecord, int, error) {
	recordsIFace, total, err := s.ldb.GetAllRecordsWithAutoId(&types.DelegatorRecord{Delegator: delegator}, limit, offset, asc)
	records := []*types.DelegatorRecord{}
	if err != nil {
		return nil, total, err
	} else {
		for _, record := range recordsIFace {
			if delegatorRecord, ok := record.(*types.DelegatorRecord); ok {
				records = append(records, delegatorRecord)
			}
		}
	}
	return records, total, nil
}

func (s *Service) GetValidatorHistory(Validator string, limit, offset int, asc bool) ([]*types.ValidatorRecord, int, error) {
	recordsIFace, total, err := s.ldb.GetAllRecordsWithAutoId(&types.ValidatorRecord{Validator: Validator}, limit, offset, asc)
	records := []*types.ValidatorRecord{}
	if err != nil {
		return nil, total, err
	} else {
		for _, record := range recordsIFace {
			if validatorRecord, ok := record.(*types.ValidatorRecord); ok {
				records = append(records, validatorRecord)
			}
		}
	}
	return records, total, nil
}

func (s *Service) GetCommissionRecord(Validator string, limit, offset int) ([]*types.CommissionRecord, int, error) {
	recordsIFace, total, err := s.ldb.GetAllRecordsWithAutoId(&types.CommissionRecord{Validator: Validator}, limit, offset, false)
	records := []*types.CommissionRecord{}
	if err != nil {
		return nil, total, err
	} else {
		for _, record := range recordsIFace {
			if commissionRecord, ok := record.(*types.CommissionRecord); ok {
				records = append(records, commissionRecord)
			}
		}
	}
	return records, total, nil
}

func (s *Service) GetStakeHistory(limit, offset int) ([]*types.StakeHistory, int, error) {
	recordsIFace, total, err := s.ldb.GetAllRecordsWithAutoId(&types.StakeHistory{}, limit, offset, false)
	records := []*types.StakeHistory{}

	stake := s.getCulStake()
	records = append(records, &types.StakeHistory{
		Amount: stake.Amount,
		Time:   time.Now(),
	})

	if err != nil {
		return nil, total, err
	} else {
		for _, record := range recordsIFace {
			if validatorRecord, ok := record.(*types.StakeHistory); ok {
				records = append(records, validatorRecord)
			}
		}
	}
	return records, total + 1, nil
}

func (s *Service) getCulStake() *types.Stake {
	stake := &types.Stake{}
	record, err := s.ldb.GetRecordByType(stake)
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
