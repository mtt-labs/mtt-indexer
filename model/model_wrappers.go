package model

import (
	"mtt-indexer/parsers"
	"mtt-indexer/types"
)

const (
	OsmosisRewardDistribution uint = iota
	TendermintLiquidityDepositCoinsToPool
	TendermintLiquidityDepositPoolCoinReceived
	TendermintLiquiditySwapTransactedCoinIn
	TendermintLiquiditySwapTransactedCoinOut
	TendermintLiquiditySwapTransactedFee
	TendermintLiquidityWithdrawPoolCoinSent
	TendermintLiquidityWithdrawCoinReceived
	TendermintLiquidityWithdrawFee
	OsmosisProtorevDeveloperRewardDistribution
)

type BlockDBWrapper struct {
	Block                         *types.Block
	BeginBlockEvents              []BlockEventDBWrapper
	EndBlockEvents                []BlockEventDBWrapper
	UniqueBlockEventTypes         map[string]types.BlockEventType
	UniqueBlockEventAttributeKeys map[string]types.BlockEventAttributeKey
}

type BlockEventDBWrapper struct {
	BlockEvent               types.BlockEvent
	Attributes               []types.BlockEventAttribute
	BlockEventParsedDatasets []parsers.BlockEventParsedData
}

// Store transactions with their messages for easy database creation
type TxDBWrapper struct {
	Tx                         types.Tx
	Messages                   []MessageDBWrapper
	UniqueMessageTypes         map[string]types.MessageType
	UniqueMessageEventTypes    map[string]types.MessageEventType
	UniqueMessageAttributeKeys map[string]types.MessageEventAttributeKey
}

type MessageDBWrapper struct {
	Message               types.Message
	MessageEvents         []MessageEventDBWrapper
	MessageParsedDatasets []parsers.MessageParsedData
}

type MessageEventDBWrapper struct {
	MessageEvent types.MessageEvent
	Attributes   []types.MessageEventAttribute
}

type DenomDBWrapper struct {
	Denom types.Denom
}
