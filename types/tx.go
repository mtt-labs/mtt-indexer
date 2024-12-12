package types

import (
	"github.com/shopspring/decimal"
)

type Tx struct {
	ID              uint
	Hash            string
	Code            uint32
	BlockID         uint
	Block           Block
	Memo            string
	SignerAddresses []Address
	Fees            []Fee
}

type FailedTx struct {
	ID      uint
	Hash    string
	BlockID uint
	Block   Block
}

type Fee struct {
	ID             uint
	TxID           uint
	Amount         decimal.Decimal
	DenominationID uint
	Denomination   Denom
	PayerAddressID uint
	PayerAddress   Address
}

// This lifecycle function ensures the on conflict statement is added for Fees which are associated to Txes by the Gorm slice association method for has_many
//func (b *Fee) BeforeCreate(tx *gorm.DB) (err error) {
//	tx.Statement.AddClause(clause.OnConflict{
//		Columns:   []clause.Column{{Name: "tx_id"}, {Name: "denomination_id"}},
//		DoUpdates: clause.AssignmentColumns([]string{"amount"}),
//	})
//	return nil
//}

type MessageType struct {
	ID          uint
	MessageType string
}

type Message struct {
	ID            uint
	TxID          uint
	Tx            Tx
	MessageTypeID uint
	MessageType   MessageType
	MessageIndex  int
	MessageBytes  []byte
}

type FailedMessage struct {
	ID           uint
	MessageIndex int
	TxID         uint
	Tx           Tx
}

type MessageEvent struct {
	ID uint
	// These fields uniquely identify every message event
	// Index refers to the position of the event in the message event array
	Index              uint64
	MessageID          uint
	Message            Message
	MessageEventTypeID uint
	MessageEventType   MessageEventType
}

type MessageEventType struct {
	ID   uint
	Type string
}

type MessageEventAttribute struct {
	ID             uint
	MessageEvent   MessageEvent
	MessageEventID uint
	Value          string
	Index          uint64
	// Keys are limited to a smallish subset of string values set by the Cosmos SDK and external modules
	// Save DB space by storing the key as a foreign key
	MessageEventAttributeKeyID uint
	MessageEventAttributeKey   MessageEventAttributeKey
}

type MessageEventAttributeKey struct {
	ID  uint
	Key string
}
