package types

import (
	"fmt"
	"time"
)

type Stake struct {
	Amount string
}

func (c *Stake) Key() string {
	return fmt.Sprintf("total_stake")
}

type StakeHistory struct {
	ID     uint64
	Amount string
	Time   time.Time
}

func (s *StakeHistory) Key() string {
	return fmt.Sprintf("StakeHistory_%d", s.ID)
}

func (s *StakeHistory) Prefix() string {
	return fmt.Sprintf("StakeHistory_")
}

func (s *StakeHistory) SetId(id uint64) {
	s.ID = id
}
