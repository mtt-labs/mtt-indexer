package service

import (
	"fmt"
	"github.com/DefiantLabs/probe/client"
	abci "github.com/cometbft/cometbft/abci/types"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	txTypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/mtt-labs/mtt-chain/crypto/ethsecp256k1"
	"github.com/syndtr/goleveldb/leveldb"
	"mtt-indexer/core"
	"mtt-indexer/db"
	"mtt-indexer/filter"
	"mtt-indexer/logger"
	"mtt-indexer/model"
	"mtt-indexer/parsers"
	"mtt-indexer/rpc"
	"mtt-indexer/types"
	"net/http"
	"sync"
	"time"
)

const RequestRetryAttempts = 10
const RequestRetryMaxWait = 10

type IndexerBlockEventData struct {
	BlockData                *ctypes.ResultBlock
	BlockResultsData         *rpc.CustomBlockResults
	BlockEventRequestsFailed bool
	GetTxsResponse           *txTypes.GetTxsEventResponse
	TxRequestsFailed         bool
	IndexBlockEvents         bool
	IndexTransactions        bool
}

type BlockEventFilterRegistries struct {
	BeginBlockEventFilterRegistry *filter.StaticBlockEventFilterRegistry
	EndBlockEventFilterRegistry   *filter.StaticBlockEventFilterRegistry
}

type ChainService struct {
	ldb   *db.LDB
	chain *types.Chain

	BlockEventFilterRegistries BlockEventFilterRegistries

	CustomBeginBlockEventParserRegistry map[string][]parsers.BlockEventParser
	CustomEndBlockEventParserRegistry   map[string][]parsers.BlockEventParser

	MessageTypeFilters []filter.MessageTypeFilter
	MessageFilters     []filter.MessageFilter

	CustomMessageParserRegistry map[string][]parsers.MessageParser
	CustomMsgTypeRegistry       map[string]sdkTypes.Msg

	cl        *client.ChainClient
	rpcClient rpc.URIClient

	txDataChan chan *DBData
}

func NewChainClient(
	chain *types.Chain,
	rpcStr string) (*client.ChainClient, error) {
	config := &client.ChainClientConfig{
		Key:            "default",
		ChainID:        chain.ChainID,
		RPCAddr:        rpcStr,
		AccountPrefix:  chain.Name,
		KeyringBackend: "test",
		Debug:          false,
		Timeout:        "60s",
		OutputFormat:   "json",
		Modules:        client.DefaultModuleBasics,
	}

	cl, err := client.NewChainClient(config, "", nil, nil)
	if err != nil {
		logger.Logger.Error(err)
		return nil, err
	}

	cl.Codec.InterfaceRegistry.RegisterImplementations((*cryptotypes.PubKey)(nil), &ethsecp256k1.PubKey{})
	cl.Codec.InterfaceRegistry.RegisterImplementations((*cryptotypes.PrivKey)(nil), &ethsecp256k1.PrivKey{})
	return cl, nil
}

func NewChainService(
	ldb *db.LDB,
	chain *types.Chain,
	cl *client.ChainClient,
) (*ChainService, error) {

	return &ChainService{
		ldb:   ldb,
		chain: chain,
		BlockEventFilterRegistries: BlockEventFilterRegistries{
			BeginBlockEventFilterRegistry: &filter.StaticBlockEventFilterRegistry{},
			EndBlockEventFilterRegistry:   &filter.StaticBlockEventFilterRegistry{},
		},
		CustomBeginBlockEventParserRegistry: nil,
		CustomEndBlockEventParserRegistry:   nil,
		cl:                                  cl,
		rpcClient: rpc.URIClient{
			Address: cl.Config.RPCAddr,
			Client:  &http.Client{},
		},
		txDataChan: make(chan *DBData, 10),
	}, nil
}

func (s *ChainService) Start(wg *sync.WaitGroup) error {
	go s.syncBlockLoop()
	go s.flushData(wg)
	return nil
}

func (s *ChainService) syncBlockLoop() {
	if err := s.syncToLatest(); err != nil {
		logger.Logger.Error("syncToLatest error %v", err)
	}

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := s.syncToLatest(); err != nil {
				logger.Logger.Error("syncToLatest error %v", err)
			}
		}
	}
}

func (s *ChainService) syncToLatest() error {
	height, err := rpc.GetLatestBlockHeight(s.cl)
	if err != nil {
		return err
	}
	for {
		if s.chain.Height != height {
			data, err := s.GetIndexerBlockEventData(s.chain.Height + 1)
			if err != nil {
				return err
			}

			err = s.processBlockData(core.HandleFailedBlock, data)
			if err != nil {
				logger.Logger.Errorf("Error processing block data: %v", err)
				return err
			}

			s.chain.Height = data.BlockData.Block.Height
		} else {
			return nil
		}
	}
}

func (s *ChainService) processBlockData(failedBlockHandler core.FailedBlockHandler, blockData *IndexerBlockEventData) error {
	block, err := core.ProcessBlock(blockData.BlockData, 1)
	if err != nil {
		logger.Logger.Error("ProcessBlock: unhandled error", err)
		return err
	}

	if blockData.IndexBlockEvents && !blockData.BlockEventRequestsFailed {
		logger.Logger.Info("Parsing block events")
		blockDBWrapper, err := core.ProcessRPCBlockResults(block, blockData.BlockResultsData, s.CustomBeginBlockEventParserRegistry, s.CustomEndBlockEventParserRegistry)
		if err != nil {
			logger.Logger.Errorf("Failed to process block events during block %d event processing, adding to failed block events table", block.Height)
		} else {
			logger.Logger.Infof("Finished parsing block event data for block %d", block.Height)

			var beginBlockFilterError error
			var endBlockFilterError error
			if s.BlockEventFilterRegistries.BeginBlockEventFilterRegistry != nil && s.BlockEventFilterRegistries.BeginBlockEventFilterRegistry.NumFilters() > 0 {
				blockDBWrapper.BeginBlockEvents, beginBlockFilterError = core.FilterRPCBlockEvents(blockDBWrapper.BeginBlockEvents, *s.BlockEventFilterRegistries.BeginBlockEventFilterRegistry)
			}

			if s.BlockEventFilterRegistries.EndBlockEventFilterRegistry != nil && s.BlockEventFilterRegistries.EndBlockEventFilterRegistry.NumFilters() > 0 {
				blockDBWrapper.EndBlockEvents, endBlockFilterError = core.FilterRPCBlockEvents(blockDBWrapper.EndBlockEvents, *s.BlockEventFilterRegistries.EndBlockEventFilterRegistry)
			}

			if beginBlockFilterError == nil && endBlockFilterError == nil {
				//blockEventsDataChan <- &BlockEventsDBData{
				//	blockDBWrapper: blockDBWrapper,
				//}
			} else {
				logger.Logger.Errorf("Failed to filter block events during block %d event processing, adding to failed block events table. Begin blocker filter error %s. End blocker filter error %s", block.Height, beginBlockFilterError, endBlockFilterError)
				failedBlockHandler(block.Height, core.FailedBlockEventHandling, err)
			}
		}
	}

	if blockData.IndexTransactions && !blockData.TxRequestsFailed {
		logger.Logger.Info("Parsing transactions")
		var txDBWrappers []model.TxDBWrapper
		var err error

		if blockData.GetTxsResponse != nil {
			logger.Logger.Debug("Processing TXs from RPC TX Search response")
			txDBWrappers, _, err = core.ProcessRPCTXs(s.cl, s.MessageTypeFilters, s.MessageFilters, blockData.GetTxsResponse, s.CustomMessageParserRegistry)
		} else if blockData.BlockResultsData != nil {
			logger.Logger.Debug("Processing TXs from BlockResults search response")
			txDBWrappers, _, err = core.ProcessRPCBlockByHeightTXs(s.cl, s.MessageTypeFilters, s.MessageFilters, blockData.BlockData, blockData.BlockResultsData, s.CustomMessageParserRegistry)
		}

		if err != nil {
			logger.Logger.Error("ProcessRpcTxs: unhandled error", err)
			failedBlockHandler(block.Height, core.UnprocessableTxError, err)
		} else {
			s.txDataChan <- &DBData{
				txDBWrappers: txDBWrappers,
				block:        block,
			}
		}

	}

	return nil
}

func (s *ChainService) flushData(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		// break out of loop once all channels are fully consumed
		if s.txDataChan == nil {
			logger.Logger.Info("DB updates complete")
			break
		}

		select {
		// read tx data from the data chan
		case data, ok := <-s.txDataChan:
			if !ok {
				s.txDataChan = nil
				continue
			}

			err := s.ldb.Transaction(
				func(ldb *db.LDB, batch *leveldb.Batch) error {
					for _, tx := range data.txDBWrappers {
						for _, message := range tx.Messages {
							if len(message.MessageParsedDatasets) != 0 {
								for _, parsedData := range message.MessageParsedDatasets {
									if parsedData.Error == nil && parsedData.Data != nil && parsedData.Parser != nil {
										combinedEventsWithAttribues := []parsers.MessageEventWithAttributes{}
										for _, event := range message.MessageEvents {
											attrs := event.Attributes
											combinedEventsWithAttribues = append(combinedEventsWithAttribues, parsers.MessageEventWithAttributes{Event: event.MessageEvent, Attributes: attrs})
										}
										err := (*parsedData.Parser).IndexMessage(ldb, batch, tx.Tx.Hash, parsedData.Data, message.Message, combinedEventsWithAttribues)
										if err != nil {
											logger.Logger.Error("Error indexing message.", err)
											return err
										}
									} else {
										logger.Logger.Infof("Error inserting message parser error.%v", parsedData)
										continue
									}
								}
							}
						}
					}

					newChain := s.chain.Clone()
					newChain.Height = data.block.Height

					err := db.StoreRecord(ldb.DB, batch, newChain)
					if err != nil {
						return err
					}
					return nil
				})
			if err != nil {
				logger.Logger.Errorf("Failed to flush data due to error %v", err)
				return
			}

			logger.Logger.Infof("Finished indexing %v TXs from block %d", len(data.txDBWrappers), data.block.Height)
		}
	}
}

func (s *ChainService) RegisterMessageTypeFilter(filter filter.MessageTypeFilter) {
	s.MessageTypeFilters = append(s.MessageTypeFilters, filter)
}

func (s *ChainService) RegisterCustomMessageParser(messageKey string, parser parsers.MessageParser) {
	if s.CustomMessageParserRegistry == nil {
		s.CustomMessageParserRegistry = make(map[string][]parsers.MessageParser)
	}
	s.CustomMessageParserRegistry[messageKey] = append(s.CustomMessageParserRegistry[messageKey], parser)
}

func (s *ChainService) RegisterCustomBeginBlockEventParser(eventKey string, parser parsers.BlockEventParser) {
	var err error
	s.CustomBeginBlockEventParserRegistry, err = customBlockEventRegistration(
		s.CustomBeginBlockEventParserRegistry,
		eventKey,
		parser,
	)
	if err != nil {
		logger.Logger.Fatal("Error registering BeginBlock custom parser", err)
	}
}

func (s *ChainService) RegisterCustomEndBlockEventParser(eventKey string, parser parsers.BlockEventParser) {
	var err error
	s.CustomBeginBlockEventParserRegistry, err = customBlockEventRegistration(
		s.CustomEndBlockEventParserRegistry,
		eventKey,
		parser,
	)
	if err != nil {
		logger.Logger.Fatal("Error registering EndBlock custom parser", err)
	}
}

func (s *ChainService) GetIndexerBlockEventData(height int64) (*IndexerBlockEventData, error) {
	blockData, err := rpc.GetBlock(s.cl, height)
	if err != nil {
		// This is the only response we continue on. If we can't get the block, we can't index anything.
		logger.Logger.Errorf("Error getting height %v from RPC. Err: %v", height, err)

		return nil, err
	}
	currentHeightIndexerData := &IndexerBlockEventData{
		BlockEventRequestsFailed: false,
		TxRequestsFailed:         false,
		IndexBlockEvents:         true,
		IndexTransactions:        true,
	}
	currentHeightIndexerData.BlockData = blockData

	if currentHeightIndexerData.IndexBlockEvents {
		bresults, err := rpc.GetBlockResultWithRetry(s.rpcClient, height, RequestRetryAttempts, RequestRetryMaxWait)

		if err != nil {
			logger.Logger.Errorf("Error getting block results for block %v from RPC. Err: %v", height, err)
			currentHeightIndexerData.BlockResultsData = nil
			currentHeightIndexerData.BlockEventRequestsFailed = true
		} else {
			bresults, err = NormalizeCustomBlockResults(bresults)
			if err != nil {
				logger.Logger.Errorf("Error normalizing block results for block %v from RPC. Err: %v", height, err)
			} else {
				currentHeightIndexerData.BlockResultsData = bresults
			}
		}
	}

	if currentHeightIndexerData.IndexTransactions {
		var txsEventResp *txTypes.GetTxsEventResponse
		var err error
		txsEventResp, err = rpc.GetTxsByBlockHeight(s.cl, height)

		if err != nil {
			// Attempt to get block results to attempt an in-app codec decode of transactions.
			if currentHeightIndexerData.BlockResultsData == nil {

				bresults, err := rpc.GetBlockResultWithRetry(s.rpcClient, height, RequestRetryAttempts, RequestRetryMaxWait)

				if err != nil {
					logger.Logger.Errorf("Error getting txs for block %v from RPC. Err: %v", height, err)

					currentHeightIndexerData.GetTxsResponse = nil
					currentHeightIndexerData.BlockResultsData = nil
					// Only set failed when we can't get the block results either.
					currentHeightIndexerData.TxRequestsFailed = true
				} else {
					bresults, err = NormalizeCustomBlockResults(bresults)
					if err != nil {
						logger.Logger.Errorf("Error normalizing block results for block %v from RPC. Err: %v", height, err)
					} else {
						currentHeightIndexerData.BlockResultsData = bresults
					}
				}

			}
		} else {
			currentHeightIndexerData.GetTxsResponse = txsEventResp
		}
	}
	return currentHeightIndexerData, nil
}

func NormalizeCustomBlockResults(blockResults *rpc.CustomBlockResults) (*rpc.CustomBlockResults, error) {
	if len(blockResults.FinalizeBlockEvents) != 0 {
		beginBlockEvents := []abci.Event{}
		endBlockEvents := []abci.Event{}

		for _, event := range blockResults.FinalizeBlockEvents {
			eventAttrs := []abci.EventAttribute{}
			isBeginBlock := false
			isEndBlock := false
			for _, attr := range event.Attributes {
				if attr.Key == "mode" {
					if attr.Value == "BeginBlock" {
						isBeginBlock = true
					} else if attr.Value == "EndBlock" {
						isEndBlock = true
					}
				} else {
					eventAttrs = append(eventAttrs, attr)
				}
			}

			switch {
			case isBeginBlock && isEndBlock:
				return nil, fmt.Errorf("finalize block event has both BeginBlock and EndBlock mode")
			case !isBeginBlock && !isEndBlock:
				return nil, fmt.Errorf("finalize block event has neither BeginBlock nor EndBlock mode")
			case isBeginBlock:
				beginBlockEvents = append(beginBlockEvents, abci.Event{Type: event.Type, Attributes: eventAttrs})
			case isEndBlock:
				endBlockEvents = append(endBlockEvents, abci.Event{Type: event.Type, Attributes: eventAttrs})
			}
		}

		blockResults.BeginBlockEvents = append(blockResults.BeginBlockEvents, beginBlockEvents...)
		blockResults.EndBlockEvents = append(blockResults.EndBlockEvents, endBlockEvents...)
	}

	return blockResults, nil
}

func customBlockEventRegistration(registry map[string][]parsers.BlockEventParser, eventKey string, parser parsers.BlockEventParser) (map[string][]parsers.BlockEventParser, error) {
	if registry == nil {
		registry = make(map[string][]parsers.BlockEventParser)
	}

	registry[eventKey] = append(registry[eventKey], parser)

	return registry, nil
}

type BlockEventsDBData struct {
	blockDBWrapper *model.BlockDBWrapper
}

type DBData struct {
	txDBWrappers []model.TxDBWrapper
	block        types.Block
}
