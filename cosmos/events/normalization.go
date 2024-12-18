package events

import (
	"fmt"
	"mtt-indexer/logger"
	"strconv"

	cometAbciTypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/types"
	txtypes "mtt-indexer/cosmos/modules/tx"
)

func NormalizedAttributesToAttributes(attrs []txtypes.Attribute) []types.Attribute {
	list := []types.Attribute{}
	for _, attr := range attrs {
		lma := types.Attribute{Key: attr.Key, Value: attr.Value}
		list = append(list, lma)
	}

	return list
}

func AttributesToNormalizedAttributes(attrs []types.Attribute) []txtypes.Attribute {
	list := []txtypes.Attribute{}
	for _, attr := range attrs {
		lma := txtypes.Attribute{Key: attr.Key, Value: attr.Value}
		list = append(list, lma)
	}

	return list
}

func EventAttributesToNormalizedAttributes(attrs []cometAbciTypes.EventAttribute) []txtypes.Attribute {
	list := []txtypes.Attribute{}
	for _, attr := range attrs {
		lma := txtypes.Attribute{Key: attr.Key, Value: attr.Value}
		list = append(list, lma)
	}

	return list
}

func StringEventstoNormalizedEvents(msgEvents types.StringEvents) (list []txtypes.LogMessageEvent) {
	for _, evt := range msgEvents {
		lme := txtypes.LogMessageEvent{Type: evt.Type, Attributes: AttributesToNormalizedAttributes(evt.Attributes)}
		list = append(list, lme)
	}

	return list
}

func toNormalizedEvents(msgEvents []cometAbciTypes.Event) (list []txtypes.LogMessageEvent) {
	for _, evt := range msgEvents {
		lme := txtypes.LogMessageEvent{Type: evt.Type, Attributes: EventAttributesToNormalizedAttributes(evt.Attributes)}
		list = append(list, lme)
	}

	return list
}

func ParseTxEventsToMessageIndexEvents(numMessages int, events []cometAbciTypes.Event) (types.ABCIMessageLogs, error) {
	parsedLogs := make(types.ABCIMessageLogs, numMessages)
	for index := range parsedLogs {
		parsedLogs[index] = types.ABCIMessageLog{
			MsgIndex: uint32(index),
		}
	}

	// TODO: Fix this to be more efficient, no need to translate multiple times to hack this together
	logMessageEvents := toNormalizedEvents(events)
	for _, event := range logMessageEvents {
		loopEvent := event
		val, err := txtypes.GetValueForAttribute("msg_index", &loopEvent)

		if err == nil && val != "" {
			msgIndex, err := strconv.Atoi(val)
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Error parsing msg_index from event: %v", err))
				return nil, err
			}

			if msgIndex >= 0 && msgIndex < len(parsedLogs) {
				parsedLogs[msgIndex].Events = append(parsedLogs[msgIndex].Events, types.StringEvent{Type: event.Type, Attributes: NormalizedAttributesToAttributes(event.Attributes)})
			}
		}
	}

	return parsedLogs, nil
}
