package types

type Address struct {
	ID      uint
	Address string `gorm:"uniqueIndex"`
}
