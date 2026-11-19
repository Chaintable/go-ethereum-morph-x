package tracers

import (
	"bytes"
	"fmt"
	"strings"

	ptypes "github.com/Chaintable/pipeline/types"
	"github.com/morph-l2/go-ethereum/common"
	"github.com/morph-l2/go-ethereum/core/types"
)

// Receipt logs are appended in execution order. The tracer emits events in call-tree order.
func validateReplayEvents(file *ptypes.BlockFile, logs []*types.Log) error {
	if len(file.Events) == 0 && len(logs) == 0 {
		return nil
	}
	parents := make(map[string]string, len(file.Traces))
	for _, trace := range file.Traces {
		parents[trace.ID] = trace.TxID
	}
	seen := make([]bool, len(logs))
	for _, event := range file.Events {
		i := event.LogIndex
		tx, ok := parents[event.ParentTraceID]
		if i < 0 || i >= int64(len(logs)) {
			return fmt.Errorf("unexpected event at log index %d (parent trace %q, tx %q)", i, event.ParentTraceID, tx)
		}
		log := logs[i]
		if seen[i] {
			return fmt.Errorf("tx %s log %d: duplicate event", log.TxHash, i)
		}
		if !ok || tx != log.TxHash.Hex() {
			return fmt.Errorf("tx %s log %d: parent trace %q resolves to tx %q", log.TxHash, i, event.ParentTraceID, tx)
		}
		if uint(i) != log.Index || !strings.EqualFold(event.Address, log.Address.Hex()) ||
			!bytes.Equal(event.Data, log.Data) || !replayTopicsMatch(event, log.Topics) {
			return fmt.Errorf("tx %s log %d: event %q differs from receipt log", log.TxHash, i, event.ID)
		}
		seen[i] = true
	}
	for i, ok := range seen {
		if !ok {
			return fmt.Errorf("tx %s log %d: missing event", logs[i].TxHash, i)
		}
	}
	return nil
}

func replayTopicsMatch(event ptypes.Event, topics []common.Hash) bool {
	if len(topics) == 0 {
		return event.Selector == "" && len(event.Topics) == 0
	}
	if event.Selector != topics[0].Hex() || len(event.Topics) != len(topics)-1 {
		return false
	}
	for i, topic := range event.Topics {
		if topic != topics[i+1].Hex() {
			return false
		}
	}
	return true
}
