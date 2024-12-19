package types

import (
	"fmt"
)

type RewardRecord struct {
	ID        uint64
	Validator string
	Amount    string
	Time      int64
}

func (v *RewardRecord) Key() string {
	return fmt.Sprintf("RewardRecord_%s_%d", v.Validator, v.ID)
}

func (v *RewardRecord) Prefix() string {
	return fmt.Sprintf("RewardRecord_%s", v.Validator)
}

func (v *RewardRecord) SetId(id uint64) {
	v.ID = id
}

type Claimed24H struct {
	Validator string
	Amount    string
}

func (d *Claimed24H) Key() string {
	return fmt.Sprintf("Claimed24H_%s", d.Validator)
}
