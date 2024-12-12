package core

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"mtt-indexer/model"

	"mtt-indexer/filter"
	"mtt-indexer/parsers"
	"mtt-indexer/rpc"
	"mtt-indexer/types"
)

func ProcessRPCBlockResults(block types.Block, blockResults *rpc.CustomBlockResults, customBeginBlockParsers map[string][]parsers.BlockEventParser, customEndBlockParsers map[string][]parsers.BlockEventParser) (*model.BlockDBWrapper, error) {
	var blockDBWrapper model.BlockDBWrapper

	blockDBWrapper.Block = &block

	blockDBWrapper.UniqueBlockEventAttributeKeys = make(map[string]types.BlockEventAttributeKey)
	blockDBWrapper.UniqueBlockEventTypes = make(map[string]types.BlockEventType)

	var err error
	blockDBWrapper.BeginBlockEvents, err = ProcessRPCBlockEvents(blockDBWrapper.Block, blockResults.BeginBlockEvents, types.BeginBlockEvent, blockDBWrapper.UniqueBlockEventTypes, blockDBWrapper.UniqueBlockEventAttributeKeys, customBeginBlockParsers)
	if err != nil {
		return nil, err
	}

	blockDBWrapper.EndBlockEvents, err = ProcessRPCBlockEvents(blockDBWrapper.Block, blockResults.EndBlockEvents, types.EndBlockEvent, blockDBWrapper.UniqueBlockEventTypes, blockDBWrapper.UniqueBlockEventAttributeKeys, customEndBlockParsers)
	if err != nil {
		return nil, err
	}

	return &blockDBWrapper, nil
}

func ProcessRPCBlockEvents(block *types.Block, blockEvents []abci.Event, blockLifecyclePosition types.BlockLifecyclePosition, uniqueEventTypes map[string]types.BlockEventType, uniqueAttributeKeys map[string]types.BlockEventAttributeKey, customParsers map[string][]parsers.BlockEventParser) ([]model.BlockEventDBWrapper, error) {
	beginBlockEvents := make([]model.BlockEventDBWrapper, len(blockEvents))

	for index, event := range blockEvents {
		eventType := types.BlockEventType{
			Type: event.Type,
		}
		beginBlockEvents[index].BlockEvent = types.BlockEvent{
			Index:             uint64(index),
			LifecyclePosition: blockLifecyclePosition,
			Block:             *block,
			BlockEventType:    eventType,
		}

		uniqueEventTypes[event.Type] = eventType

		beginBlockEvents[index].Attributes = make([]types.BlockEventAttribute, len(event.Attributes))

		for attrIndex, attribute := range event.Attributes {

			var value string
			var keyItem string
			value = attribute.Value
			keyItem = attribute.Key

			key := types.BlockEventAttributeKey{
				Key: keyItem,
			}

			beginBlockEvents[index].Attributes[attrIndex] = types.BlockEventAttribute{
				Value:                  value,
				BlockEventAttributeKey: key,
				Index:                  uint64(attrIndex),
			}

			uniqueAttributeKeys[key.Key] = key

		}

		if customParsers != nil {
			if customBlockEventParsers, ok := customParsers[event.Type]; ok {
				for index, customParser := range customBlockEventParsers {
					// We deliberately ignore the error here, as we want to continue processing the block events even if a custom parsers fails
					parsedData, err := customParser.ParseBlockEvent(event)
					beginBlockEvents[index].BlockEventParsedDatasets = append(beginBlockEvents[index].BlockEventParsedDatasets, parsers.BlockEventParsedData{
						Data:   parsedData,
						Error:  err,
						Parser: &customBlockEventParsers[index],
					})
				}
			}
		}

	}

	return beginBlockEvents, nil
}

func FilterRPCBlockEvents(blockEvents []model.BlockEventDBWrapper, filterRegistry filter.StaticBlockEventFilterRegistry) ([]model.BlockEventDBWrapper, error) {
	// If there are no filters, just return the block events
	if len(filterRegistry.BlockEventFilters) == 0 && len(filterRegistry.RollingWindowEventFilters) == 0 {
		return blockEvents, nil
	}

	filterIndexes := make(map[int]bool)

	// If filters are defined, we treat filters as a whitelist, and only include block events that match the filters and are allowed
	// Filters are evaluated in order, and the first filter that matches is the one that is used. Single block event filters are preferred in ordering.
	for index, blockEvent := range blockEvents {
		filterEvent := filter.EventData{
			Event:      blockEvent.BlockEvent,
			Attributes: blockEvent.Attributes,
		}

		for _, filter := range filterRegistry.BlockEventFilters {
			patternMatch, err := filter.EventMatches(filterEvent)
			if err != nil {
				return nil, err
			}
			if patternMatch {
				filterIndexes[index] = filter.IncludeMatch()
			}
		}

		for _, rollingWindowFilter := range filterRegistry.RollingWindowEventFilters {
			if index+rollingWindowFilter.RollingWindowLength() <= len(blockEvents) {
				lastIndex := index + rollingWindowFilter.RollingWindowLength()
				blockEventSlice := blockEvents[index:lastIndex]

				filterEvents := make([]filter.EventData, len(blockEventSlice))

				for index, blockEvent := range blockEventSlice {
					filterEvents[index] = filter.EventData{
						Event:      blockEvent.BlockEvent,
						Attributes: blockEvent.Attributes,
					}
				}

				patternMatches, err := rollingWindowFilter.EventsMatch(filterEvents)
				if err != nil {
					return nil, err
				}

				if patternMatches {
					for i := index; i < lastIndex; i++ {
						filterIndexes[i] = rollingWindowFilter.IncludeMatches()
					}
				}
			}
		}
	}

	// Filter the block events based on the indexes that matched the registered patterns
	filteredBlockEvents := make([]model.BlockEventDBWrapper, 0)

	for index, blockEvent := range blockEvents {
		if filterIndexes[index] {
			filteredBlockEvents = append(filteredBlockEvents, blockEvent)
		}
	}

	return filteredBlockEvents, nil
}
