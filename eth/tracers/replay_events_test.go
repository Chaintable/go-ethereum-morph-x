package tracers

import (
	"strings"
	"testing"

	ptypes "github.com/Chaintable/pipeline/types"
	"github.com/morph-l2/go-ethereum/common"
	"github.com/morph-l2/go-ethereum/core/types"
)

func TestValidateReplayEvents(t *testing.T) {
	txA, txB := common.BytesToHash([]byte{1}), common.BytesToHash([]byte{2})
	topic := common.BytesToHash([]byte{3})
	logs := []*types.Log{
		{TxHash: txA, Index: 0},
		{TxHash: txB, Index: 1, Topics: []common.Hash{topic, txB}, Data: []byte{1}},
	}
	tests := []struct {
		name string
		edit func(*ptypes.BlockFile)
		want string
	}{
		{"out of order and LOG0", func(*ptypes.BlockFile) {}, ""},
		{"missing", func(f *ptypes.BlockFile) { f.Events = f.Events[:1] }, "missing event"},
		{"extra", func(f *ptypes.BlockFile) { f.Events[0].LogIndex = 2 }, "unexpected event"},
		{"negative index", func(f *ptypes.BlockFile) { f.Events[0].LogIndex = -1 }, "unexpected event"},
		{"duplicate", func(f *ptypes.BlockFile) { f.Events[1] = f.Events[0] }, "duplicate event"},
		{"orphan", func(f *ptypes.BlockFile) { f.Events[0].ParentTraceID = "unknown" }, "parent trace"},
		{"wrong transaction", func(f *ptypes.BlockFile) { f.Events[0].ParentTraceID = "a" }, "parent trace"},
		{"address", func(f *ptypes.BlockFile) { f.Events[0].Address = "" }, "differs"},
		{"data", func(f *ptypes.BlockFile) { f.Events[0].Data = nil }, "differs"},
		{"selector", func(f *ptypes.BlockFile) { f.Events[0].Selector = "" }, "differs"},
		{"topic", func(f *ptypes.BlockFile) { f.Events[0].Topics[0] = topic.Hex() }, "differs"},
		{"LOG0 with topic", func(f *ptypes.BlockFile) { f.Events[1].Topics = []string{topic.Hex()} }, "differs"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := &ptypes.BlockFile{
				Traces: []ptypes.Trace{{ID: "a", TxID: txA.Hex()}, {ID: "b", TxID: txB.Hex()}},
				Events: []ptypes.Event{
					{ParentTraceID: "b", LogIndex: 1, Address: logs[1].Address.Hex(), Selector: topic.Hex(), Topics: []string{txB.Hex()}, Data: []byte{1}},
					{ParentTraceID: "a", LogIndex: 0, Address: logs[0].Address.Hex()},
				},
				ErrorEvents: []ptypes.Event{{LogIndex: 0}},
			}
			test.edit(file)
			err := validateReplayEvents(file, logs)
			if test.want == "" && err != nil || test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}
}
