package types

//import (
//	"mtt-indexer/parsers"
//)
//
//const (
//	OsmosisRewardDistribution uint = iota
//	TendermintLiquidityDepositCoinsToPool
//	TendermintLiquidityDepositPoolCoinReceived
//	TendermintLiquiditySwapTransactedCoinIn
//	TendermintLiquiditySwapTransactedCoinOut
//	TendermintLiquiditySwapTransactedFee
//	TendermintLiquidityWithdrawPoolCoinSent
//	TendermintLiquidityWithdrawCoinReceived
//	TendermintLiquidityWithdrawFee
//	OsmosisProtorevDeveloperRewardDistribution
//)
//
//type BlockDBWrapper struct {
//	Block                         *Block
//	BeginBlockEvents              []BlockEventDBWrapper
//	EndBlockEvents                []BlockEventDBWrapper
//	UniqueBlockEventTypes         map[string]BlockEventType
//	UniqueBlockEventAttributeKeys map[string]BlockEventAttributeKey
//}
//
//type BlockEventDBWrapper struct {
//	BlockEvent               BlockEvent
//	Attributes               []BlockEventAttribute
//	BlockEventParsedDatasets []parsers.BlockEventParsedData
//}
//
//// Store transactions with their messages for easy database creation
//type TxDBWrapper struct {
//	Tx                         Tx
//	Messages                   []MessageDBWrapper
//	UniqueMessageTypes         map[string]MessageType
//	UniqueMessageEventTypes    map[string]MessageEventType
//	UniqueMessageAttributeKeys map[string]MessageEventAttributeKey
//}
//
//type MessageDBWrapper struct {
//	Message               Message
//	MessageEvents         []MessageEventDBWrapper
//	MessageParsedDatasets []parsers.MessageParsedData
//}
//
//type MessageEventDBWrapper struct {
//	MessageEvent MessageEvent
//	Attributes   []MessageEventAttribute
//}
//
//type DenomDBWrapper struct {
//	Denom Denom
//}
