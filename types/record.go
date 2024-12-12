package types

import (
	sdkmath "cosmossdk.io/math"
	"fmt"
	"time"
)

type DelegationType uint8

const (
	Delegate DelegationType = iota
	Undelegate
	Claim
	CancelUnbonding
	Redelegate
)

type DbRecord interface {
	Key() string
}

type DbRecordAutoId interface {
	DbRecord
	Prefix() string
	SetId(uint64)
}

type Record struct {
	Status uint8 // 0: ongoing      1: end
}

type ValidatorRecord struct {
	ID             uint64
	Delegator      string
	Validator      string
	Amount         string
	Denom          string
	TxHash         string
	DelegationType DelegationType //true add false rm
	DelegationTime time.Time
}

type RedelegateRecord struct {
	ID             uint64
	Delegator      string
	Src            string
	Dst            string
	Amount         string
	Denom          string
	TxHash         string
	DelegationType DelegationType //true add false rm
	DelegationTime time.Time
}

func (v *ValidatorRecord) Key() string {
	return fmt.Sprintf("ValidatorRecord_%s_%d", v.Validator, v.ID)
}

func (v *ValidatorRecord) Prefix() string {
	return fmt.Sprintf("ValidatorRecord_%s", v.Validator)
}

func (v *ValidatorRecord) SetId(id uint64) {
	v.ID = id
}

func (v *ValidatorRecord) ToDelegate() *DelegatorRecord {
	return &DelegatorRecord{
		ID:             0,
		Delegator:      v.Delegator,
		Validator:      v.Validator,
		Amount:         v.Amount,
		Denom:          v.Denom,
		TxHash:         v.TxHash,
		DelegationType: v.DelegationType,
		DelegationTime: v.DelegationTime,
	}
}

type DelegatorRecord struct {
	ID             uint64
	Delegator      string
	Validator      string
	Amount         string
	Denom          string
	TxHash         string
	DelegationType DelegationType //true add false rm
	DelegationTime time.Time
}

func (v *DelegatorRecord) Key() string {
	return fmt.Sprintf("DelegatorRecord_%s_%d", v.Delegator, v.ID)
}

func (v *DelegatorRecord) Prefix() string {
	return fmt.Sprintf("DelegatorRecord_%s", v.Delegator)
}

func (v *DelegatorRecord) SetId(id uint64) {
	v.ID = id
}

func (v *DelegatorRecord) ToValidator() *ValidatorRecord {
	return &ValidatorRecord{
		Delegator:      v.Delegator,
		Validator:      v.Validator,
		Amount:         v.Amount,
		Denom:          v.Denom,
		TxHash:         v.TxHash,
		DelegationType: v.DelegationType,
		DelegationTime: v.DelegationTime,
	}
}

type CommissionRecord struct {
	ID         uint64
	Validator  string
	Commission float64
	Time       time.Time
}

func (c *CommissionRecord) Key() string {
	return fmt.Sprintf("CommissionRecord_%s_%d", c.Validator, c.ID)
}

func (c *CommissionRecord) Prefix() string {
	return fmt.Sprintf("CommissionRecord_%s", c.Validator)
}

func (c *CommissionRecord) SetId(id uint64) {
	c.ID = id
}

type DelegatorOutList struct {
	Delegator  string
	Validators []string
	Amounts    []string
	Denom      string
}

func (d *DelegatorOutList) Key() string {
	return fmt.Sprintf("DelegatorOutList_%s", d.Delegator)
}

func (d *DelegatorOutList) AddValidatorRecord(record ValidatorRecord, delta bool) {
	contain := false
	for index, v := range d.Validators {
		if v == record.Validator {
			contain = true
			amount, _ := sdkmath.NewIntFromString(d.Amounts[index])
			vAmount, _ := sdkmath.NewIntFromString(d.Amounts[index])
			if delta {
				amount = amount.Add(vAmount)
			} else {
				amount = amount.Sub(vAmount)
			}

			d.Amounts[index] = amount.String()
		}
	}
	if !contain {
		d.Validators = append(d.Validators, record.Validator)
		d.Amounts = append(d.Amounts, record.Amount)
	}
}
