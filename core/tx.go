package core

import (
	"encoding/hex"
	"fmt"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"math/big"
	txtypes "mtt-indexer/cosmos/modules/tx"
	"mtt-indexer/filter"
	"mtt-indexer/logger"
	"mtt-indexer/model"
	"mtt-indexer/parsers"
	"mtt-indexer/rpc"
	"mtt-indexer/types"
	"mtt-indexer/util"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/DefiantLabs/probe/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptoTypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmosTx "github.com/cosmos/cosmos-sdk/types/tx"

	indexerEvents "mtt-indexer/cosmos/events"
)

func getUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func ProcessRPCBlockByHeightTXs(cl *client.ChainClient, messageTypeFilters []filter.MessageTypeFilter, messageFilters []filter.MessageFilter, blockResults *coretypes.ResultBlock, resultBlockRes *rpc.CustomBlockResults, customParsers map[string][]parsers.MessageParser) ([]model.TxDBWrapper, *time.Time, error) {
	if len(blockResults.Block.Txs) != len(resultBlockRes.TxsResults) {
		logger.Logger.Fatalf("blockResults & resultBlockRes: different length")
	}

	blockTime := &blockResults.Block.Time
	blockTimeStr := blockTime.Format(time.RFC3339)
	var currTxDbWrappers []model.TxDBWrapper

	for txIdx, tendermintTx := range blockResults.Block.Txs {
		txResult := resultBlockRes.TxsResults[txIdx]

		// Indexer types only used by the indexer app (similar to the cosmos types)
		var indexerMergedTx txtypes.MergedTx
		var indexerTx txtypes.IndexerTx
		var txBody txtypes.Body
		var currMessages []sdktypes.Msg
		var currLogMsgs []txtypes.LogMessage

		txDecoder := cl.Codec.TxConfig.TxDecoder()

		txBasic, err := txDecoder(tendermintTx)
		var txFull *cosmosTx.Tx
		if err != nil {
			txBasic, err = InAppTxDecoder(cl.Codec)(tendermintTx)
			if err != nil {
				return nil, blockTime, fmt.Errorf("ProcessRPCBlockByHeightTXs: TX cannot be parsed from block %v. This is usually a proto definition error. Err: %v", blockResults.Block.Height, err)
			}
			txFull = txBasic.(*cosmosTx.Tx)
		} else {
			// This is a hack, but as far as I can tell necessary. "wrapper" struct is private in Cosmos SDK.
			field := reflect.ValueOf(txBasic).Elem().FieldByName("tx")
			iTx := getUnexportedField(field)
			txFull = iTx.(*cosmosTx.Tx)
		}

		logs := sdktypes.ABCIMessageLogs{}

		// Failed TXs do not have proper JSON in the .Log field, causing ParseABCILogs to fail to unmarshal the logs
		// We can entirely ignore failed TXs in downstream parsers, because according to the Cosmos specification, a single failed message in a TX fails the whole TX
		if txResult.Code == 0 {
			logs, err = sdktypes.ParseABCILogs(txResult.Log)
			if err != nil {
				logs, err = indexerEvents.ParseTxEventsToMessageIndexEvents(len(txFull.Body.Messages), txResult.Events)
			}
		} else {
			err = nil
		}

		if err != nil {
			logger.Logger.Errorf("Error parsing events to message index events to normalize: %v", err)
			return nil, blockTime, fmt.Errorf("logs could not be parsed")
		}

		txHash := tendermintTx.Hash()

		var messagesRaw [][]byte
		var messageTypeURLs []string

		// Get the Messages and Message Logs
		for msgIdx := range txFull.Body.Messages {

			shouldIndex, err := messageTypeShouldIndex(txFull.Body.Messages[msgIdx].TypeUrl, messageTypeFilters, customParsers)
			if err != nil {
				return nil, blockTime, err
			}

			messageTypeURLs = append(messageTypeURLs, txFull.Body.Messages[msgIdx].TypeUrl)

			if !shouldIndex {
				logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Skipping msg of type '%v' due to message type filter.", blockResults.Block.Height, tendermintHashToHex(txHash), txFull.Body.Messages[msgIdx].TypeUrl))
				currMessages = append(currMessages, nil)
				currLogMsgs = append(currLogMsgs, txtypes.LogMessage{
					MessageIndex: msgIdx,
				})
				messagesRaw = append(messagesRaw, nil)
				continue
			}

			currMsg := txFull.Body.Messages[msgIdx].GetCachedValue()

			if currMsg != nil {
				msg := currMsg.(sdktypes.Msg)
				msgEvents := sdktypes.StringEvents{}
				if txResult.Code == 0 {
					msgEvents = logs[msgIdx].Events
				}

				currTxLog := txtypes.LogMessage{
					MessageIndex: msgIdx,
					Events:       indexerEvents.StringEventstoNormalizedEvents(msgEvents),
				}

				// Next filter on the message itself if there are any filters
				if len(messageFilters) != 0 {
					for _, filter := range messageFilters {
						shouldIndex = filter.ShouldIndex(msg, currTxLog)
						if shouldIndex {
							break
						}
					}
				}

				if !shouldIndex {
					logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Skipping msg of type '%v' due to custom message filter.", blockResults.Block.Height, tendermintHashToHex(txHash), txFull.Body.Messages[msgIdx].TypeUrl))
					currMessages = append(currMessages, nil)
					currLogMsgs = append(currLogMsgs, txtypes.LogMessage{
						MessageIndex: msgIdx,
					})
					messagesRaw = append(messagesRaw, nil)
					continue
				}

				messagesRaw = append(messagesRaw, txFull.Body.Messages[msgIdx].Value)
				currMessages = append(currMessages, msg)
				currLogMsgs = append(currLogMsgs, currTxLog)
			} else {
				return nil, blockTime, fmt.Errorf("tx message could not be processed")
			}
		}

		hexTxHash := tendermintHashToHex(txHash)

		txBody.Messages = currMessages
		indexerTx.Body = txBody
		indexerTxResp := txtypes.Response{
			TxHash:    hexTxHash,
			Height:    fmt.Sprintf("%d", blockResults.Block.Height),
			TimeStamp: blockTimeStr,
			RawLog:    txResult.Log,
			Log:       currLogMsgs,
			Code:      txResult.Code,
		}

		indexerTx.AuthInfo = *txFull.AuthInfo
		indexerMergedTx.TxResponse = indexerTxResp
		indexerMergedTx.Tx = indexerTx
		indexerMergedTx.Tx.AuthInfo = *txFull.AuthInfo

		processedTx, _, err := ProcessTx(indexerMergedTx, messagesRaw, messageTypeURLs, customParsers)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}

		if len(processedTx.Messages) == 0 {
			logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Skipping empty transaction.", blockResults.Block.Height, hexTxHash))
			continue
		}

		logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Processing transaction with %d messages.", blockResults.Block.Height, hexTxHash, len(processedTx.Messages)))

		filteredSigners := []sdktypes.AccAddress{}
		for _, filteredMessage := range txBody.Messages {
			if filteredMessage != nil {
				filteredSigners = append(filteredSigners, filteredMessage.GetSigners()...)
			}
		}

		signers, err := ProcessSigners(cl, txFull.AuthInfo, filteredSigners)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}

		processedTx.Tx.SignerAddresses = signers

		fees, err := ProcessFees(indexerTx.AuthInfo, signers)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}

		processedTx.Tx.Fees = fees
		processedTx.Tx.Memo = txFull.Body.Memo

		currTxDbWrappers = append(currTxDbWrappers, processedTx)
	}

	return currTxDbWrappers, blockTime, nil
}

func tendermintHashToHex(hash []byte) string {
	return strings.ToUpper(hex.EncodeToString(hash))
}

// ProcessRPCTXs - Given an RPC response, build out the more specific data used by the parser.
func ProcessRPCTXs(cl *client.ChainClient, messageTypeFilters []filter.MessageTypeFilter, messageFilters []filter.MessageFilter, txEventResp *cosmosTx.GetTxsEventResponse, customParsers map[string][]parsers.MessageParser) ([]model.TxDBWrapper, *time.Time, error) {
	var currTxDbWrappers []model.TxDBWrapper
	var blockTime *time.Time

	for txIdx := range txEventResp.Txs {
		// Indexer types only used by the indexer app (similar to the cosmos types)
		var indexerMergedTx txtypes.MergedTx
		var indexerTx txtypes.IndexerTx
		var txBody txtypes.Body
		var currMessages []sdktypes.Msg
		var currLogMsgs []txtypes.LogMessage
		var messagesRaw [][]byte
		var messageTypeURLs []string

		currTx := txEventResp.Txs[txIdx]
		currTxResp := txEventResp.TxResponses[txIdx]

		if len(currTxResp.Logs) == 0 && len(currTxResp.Events) != 0 {
			// We have a version of Cosmos SDK that removed the Logs field from the TxResponse, we need to parse the events into message index logs
			parsedLogs, err := indexerEvents.ParseTxEventsToMessageIndexEvents(len(currTx.Body.Messages), currTxResp.Events)
			if err != nil {
				logger.Logger.Errorf("Error parsing events to message index events to normalize: %v", err)
				return nil, blockTime, err
			}

			currTxResp.Logs = parsedLogs
		}

		// Get the Messages and Message Logs
		for msgIdx := range currTx.Body.Messages {

			shouldIndex, err := messageTypeShouldIndex(currTx.Body.Messages[msgIdx].TypeUrl, messageTypeFilters, customParsers)
			if err != nil {
				return nil, blockTime, err
			}

			messageTypeURLs = append(messageTypeURLs, currTx.Body.Messages[msgIdx].TypeUrl)

			if !shouldIndex {
				logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Skipping msg of type '%v' due to message type filter.", currTxResp.Height, currTxResp.TxHash, currTx.Body.Messages[msgIdx].TypeUrl))
				currMessages = append(currMessages, nil)
				currLogMsgs = append(currLogMsgs, txtypes.LogMessage{
					MessageIndex: msgIdx,
				})
				messagesRaw = append(messagesRaw, nil)
				continue
			}

			currMsg := currTx.Body.Messages[msgIdx].GetCachedValue()
			messagesRaw = append(messagesRaw, currTx.Body.Messages[msgIdx].Value)

			// If we reached here, unpacking the entire TX raw was not successful
			// Attempt to unpack the message individually.
			if currMsg == nil {
				var currMsgUnpack sdktypes.Msg
				err := cl.Codec.InterfaceRegistry.UnpackAny(currTx.Body.Messages[msgIdx], &currMsgUnpack)
				if err != nil || currMsgUnpack == nil {
					return nil, blockTime, fmt.Errorf("tx message could not be processed. Unpacking protos failed and CachedValue is not present. TX Hash: %s, Msg type: %s, Msg index: %d, Code: %d",
						currTxResp.TxHash,
						currTx.Body.Messages[msgIdx].TypeUrl,
						msgIdx,
						currTxResp.Code,
					)
				}
				currMsg = currMsgUnpack
			}

			if currMsg != nil {
				msg := currMsg.(sdktypes.Msg)
				msgEvents := currTxResp.Logs[msgIdx].Events
				currTxLog := txtypes.LogMessage{
					MessageIndex: msgIdx,
					Events:       indexerEvents.StringEventstoNormalizedEvents(msgEvents),
				}

				if len(messageFilters) != 0 {
					for _, filter := range messageFilters {
						shouldIndex = filter.ShouldIndex(msg, currTxLog)
						if shouldIndex {
							break
						}
					}
				}

				if !shouldIndex {
					logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Skipping msg of type '%v' due to custom message filter.", currTxResp.Height, currTxResp.TxHash, currTx.Body.Messages[msgIdx].TypeUrl))
					currMessages = append(currMessages, nil)
					currLogMsgs = append(currLogMsgs, txtypes.LogMessage{
						MessageIndex: msgIdx,
					})
					messagesRaw = append(messagesRaw, nil)
					continue
				}

				currMessages = append(currMessages, msg)
				currLogMsgs = append(currLogMsgs, currTxLog)
			}
		}

		txBody.Messages = currMessages
		indexerTx.Body = txBody

		indexerTxResp := txtypes.Response{
			TxHash:    currTxResp.TxHash,
			Height:    fmt.Sprintf("%d", currTxResp.Height),
			TimeStamp: currTxResp.Timestamp,
			RawLog:    currTxResp.RawLog,
			Log:       currLogMsgs,
			Code:      currTxResp.Code,
		}

		indexerTx.AuthInfo = *currTx.AuthInfo
		indexerMergedTx.TxResponse = indexerTxResp
		indexerMergedTx.Tx = indexerTx
		indexerMergedTx.Tx.AuthInfo = *currTx.AuthInfo

		processedTx, txTime, err := ProcessTx(indexerMergedTx, messagesRaw, messageTypeURLs, customParsers)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}

		if len(processedTx.Messages) == 0 {
			logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Skipping empty transaction.", currTxResp.Height, currTxResp.TxHash))
			continue
		}

		logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Processing transaction with %d messages.", currTxResp.Height, currTxResp.TxHash, len(processedTx.Messages)))

		if blockTime == nil {
			blockTime = &txTime
		}

		filteredSigners := []sdktypes.AccAddress{}
		for _, filteredMessage := range txBody.Messages {
			if filteredMessage != nil {
				filteredSigners = append(filteredSigners, filteredMessage.GetSigners()...)
			}
		}

		err = currTx.AuthInfo.UnpackInterfaces(cl.Codec.InterfaceRegistry)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}

		signers, err := ProcessSigners(cl, currTx.AuthInfo, filteredSigners)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}
		processedTx.Tx.SignerAddresses = signers

		fees, err := ProcessFees(indexerTx.AuthInfo, signers)
		if err != nil {
			return currTxDbWrappers, blockTime, err
		}

		processedTx.Tx.Fees = fees
		processedTx.Tx.Memo = currTx.Body.Memo

		currTxDbWrappers = append(currTxDbWrappers, processedTx)
	}

	return currTxDbWrappers, blockTime, nil
}

func messageTypeShouldIndex(messageType string, filters []filter.MessageTypeFilter, customParsers map[string][]parsers.MessageParser) (bool, error) {
	// Always index if a custom parser for the message type is present
	if len(customParsers) != 0 {
		if customParsers[messageType] != nil {
			return true, nil
		}
	}

	if len(filters) != 0 {
		filterData := filter.MessageTypeData{
			MessageType: messageType,
		}

		matches := false
		for _, messageTypeFilter := range filters {
			typeMatch, err := messageTypeFilter.MessageTypeMatches(filterData)
			if err != nil {
				return false, err
			}

			// If any match is marked to ignore, always ignore
			if typeMatch && messageTypeFilter.Ignore() {
				return false, nil
			} else if typeMatch {
				matches = true
			}
		}

		return matches, nil
	}

	return true, nil
}

func ProcessTx(tx txtypes.MergedTx, messagesRaw [][]byte, messageTypeURLs []string, customParsers map[string][]parsers.MessageParser) (txDBWapper model.TxDBWrapper, txTime time.Time, err error) {
	txTime, err = time.Parse(time.RFC3339, tx.TxResponse.TimeStamp)
	if err != nil {
		logger.Logger.Error("Error parsing tx timestamp.", err)
		return txDBWapper, txTime, err
	}

	code := tx.TxResponse.Code

	var messages []model.MessageDBWrapper

	uniqueMessageTypes := make(map[string]types.MessageType)
	uniqueEventTypes := make(map[string]types.MessageEventType)
	uniqueEventAttributeKeys := make(map[string]types.MessageEventAttributeKey)
	// non-zero code means the Tx was unsuccessful. We will still need to account for fees in both cases though.
	if code == 0 {
		for messageIndex, message := range tx.Tx.Body.Messages {
			if message != nil {
				messageLog := txtypes.GetMessageLogForIndex(tx.TxResponse.Log, messageIndex)
				messageType, currMessageDBWrapper := ProcessMessage(messageIndex, message, messageTypeURLs[messageIndex], messageLog, uniqueEventTypes, uniqueEventAttributeKeys)
				currMessageDBWrapper.Message.Tx.Block.TimeStamp = txTime
				currMessageDBWrapper.Message.MessageBytes = messagesRaw[messageIndex]
				uniqueMessageTypes[messageType] = currMessageDBWrapper.Message.MessageType
				logger.Logger.Debug(fmt.Sprintf("[Block: %v] [TX: %v] Found msg of type '%v'.", tx.TxResponse.Height, tx.TxResponse.TxHash, messageType))

				if customParsers != nil {
					if customMessageParsers, ok := customParsers[messageType]; ok {
						for index, customParser := range customMessageParsers {
							// We deliberately ignore the error here, as we want to continue processing the message even if a custom parser fails
							parsedData, err := customParser.ParseMessage(message, messageLog)

							currMessageDBWrapper.MessageParsedDatasets = append(currMessageDBWrapper.MessageParsedDatasets, parsers.MessageParsedData{
								Data:   parsedData,
								Error:  err,
								Parser: &customMessageParsers[index],
							})
						}
					}
				}

				messages = append(messages, currMessageDBWrapper)
			}
		}
	}
	txDBWapper.Tx = types.Tx{Hash: tx.TxResponse.TxHash, Code: code}
	txDBWapper.Messages = messages
	txDBWapper.UniqueMessageTypes = uniqueMessageTypes
	txDBWapper.UniqueMessageAttributeKeys = uniqueEventAttributeKeys
	txDBWapper.UniqueMessageEventTypes = uniqueEventTypes

	return txDBWapper, txTime, nil
}

// Processes signers in a deterministic order.
// 1. Processes signers from the auth info
// 2. Processes signers from the signers array
// 3. Processes the fee payer
func ProcessSigners(cl *client.ChainClient, authInfo *cosmosTx.AuthInfo, messageSigners []sdktypes.AccAddress) ([]types.Address, error) {
	// For unique checks
	signerAddressMap := make(map[string]types.Address)
	// For deterministic output of signer values
	var signerAddressArray []types.Address

	// If there is a signer info, get the addresses from the keys add it to the list of signers
	for _, signerInfo := range authInfo.SignerInfos {
		if signerInfo.PublicKey != nil {
			pubKey, err := cl.Codec.InterfaceRegistry.Resolve(signerInfo.PublicKey.TypeUrl)
			if err != nil {
				return nil, err
			}
			err = cl.Codec.InterfaceRegistry.UnpackAny(signerInfo.PublicKey, &pubKey)
			if err != nil {
				return nil, err
			}

			multisigKey, ok := pubKey.(*multisig.LegacyAminoPubKey)

			if ok {
				for _, key := range multisigKey.GetPubKeys() {
					address := sdktypes.AccAddress(key.Address().Bytes()).String()
					if _, ok := signerAddressMap[address]; !ok {
						signerAddressArray = append(signerAddressArray, types.Address{Address: address})
					}
					signerAddressMap[address] = types.Address{Address: address}
				}
			} else {
				castPubKey, ok := pubKey.(cryptoTypes.PubKey)
				if !ok {
					return nil, err
				}

				address := sdktypes.AccAddress(castPubKey.Address().Bytes()).String()
				if _, ok := signerAddressMap[address]; !ok {
					signerAddressArray = append(signerAddressArray, types.Address{Address: address})
				}
				signerAddressMap[address] = types.Address{Address: address}
			}

		}
	}

	for _, signer := range messageSigners {
		addressStr := signer.String()
		if _, ok := signerAddressMap[addressStr]; !ok {
			signerAddressArray = append(signerAddressArray, types.Address{Address: addressStr})
		}
		signerAddressMap[addressStr] = types.Address{Address: addressStr}
	}

	// If there is a fee payer, add it to the list of signers
	if authInfo.Fee.GetPayer() != "" {
		if _, ok := signerAddressMap[authInfo.Fee.GetPayer()]; !ok {
			signerAddressArray = append(signerAddressArray, types.Address{Address: authInfo.Fee.GetPayer()})
		}
		signerAddressMap[authInfo.Fee.GetPayer()] = types.Address{Address: authInfo.Fee.GetPayer()}
	}

	return signerAddressArray, nil
}

// Processes fees into model form, applying denoms and addresses to them
func ProcessFees(authInfo cosmosTx.AuthInfo, signers []types.Address) ([]types.Fee, error) {
	feeCoins := authInfo.Fee.Amount
	payer := authInfo.Fee.GetPayer()
	fees := []types.Fee{}

	for _, coin := range feeCoins {
		zeroFee := big.NewInt(0)

		if zeroFee.Cmp(coin.Amount.BigInt()) != 0 {
			amount := util.ToNumeric(coin.Amount.BigInt())
			denom := types.Denom{Base: coin.Denom}

			payerAddr := types.Address{}
			if payer != "" {
				payerAddr.Address = payer
			} else if len(signers) > 0 {
				payerAddr = signers[0]
			}

			fees = append(fees, types.Fee{Amount: amount, Denomination: denom, PayerAddress: payerAddr})
		}
	}

	return fees, nil
}

func ProcessMessage(messageIndex int, message sdktypes.Msg, messageTypeURL string, messageLog *txtypes.LogMessage, uniqueEventTypes map[string]types.MessageEventType, uniqueEventAttributeKeys map[string]types.MessageEventAttributeKey) (string, model.MessageDBWrapper) {
	var currMessage types.Message
	var currMessageType types.MessageType
	currMessage.MessageIndex = messageIndex

	// Get the message log that corresponds to the current message
	var currMessageDBWrapper model.MessageDBWrapper

	currMessageType.MessageType = messageTypeURL
	currMessage.MessageType = currMessageType
	currMessageDBWrapper.Message = currMessage

	for eventIndex, event := range messageLog.Events {
		uniqueEventTypes[event.Type] = types.MessageEventType{Type: event.Type}

		var currMessageEvent model.MessageEventDBWrapper
		currMessageEvent.MessageEvent = types.MessageEvent{
			MessageEventType: uniqueEventTypes[event.Type],
			Index:            uint64(eventIndex),
		}
		var currMessageEventAttributes []types.MessageEventAttribute
		for attributeIndex, attribute := range event.Attributes {
			uniqueEventAttributeKeys[attribute.Key] = types.MessageEventAttributeKey{Key: attribute.Key}

			currMessageEventAttributes = append(currMessageEventAttributes, types.MessageEventAttribute{
				Value:                    attribute.Value,
				MessageEventAttributeKey: uniqueEventAttributeKeys[attribute.Key],
				Index:                    uint64(attributeIndex),
			})
		}

		currMessageEvent.Attributes = currMessageEventAttributes
		currMessageDBWrapper.MessageEvents = append(currMessageDBWrapper.MessageEvents, currMessageEvent)
	}
	return currMessageType.MessageType, currMessageDBWrapper
}
