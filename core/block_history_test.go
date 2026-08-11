// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package core

import (
	"math/big"
	"testing"

	"github.com/morph-l2/go-ethereum/core/rawdb"
	"github.com/morph-l2/go-ethereum/core/types"
	"github.com/morph-l2/go-ethereum/ethdb/memorydb"
)

func TestPruneBlockHistoryWaitsForIndexes(t *testing.T) {
	db, err := rawdb.NewDatabaseWithFreezer(memorydb.New(), t.TempDir(), "", false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	blocks := make([]*types.Block, 0, 10)
	for i := uint64(0); i < 10; i++ {
		blocks = append(blocks, types.NewBlockWithHeader(&types.Header{
			Number:      new(big.Int).SetUint64(i),
			Difficulty:  big.NewInt(1),
			UncleHash:   types.EmptyUncleHash,
			TxHash:      types.EmptyRootHash,
			ReceiptHash: types.EmptyRootHash,
		}))
	}
	if _, err := rawdb.WriteAncientBlocks(db, blocks, make([]types.Receipts, len(blocks)), big.NewInt(1)); err != nil {
		t.Fatal(err)
	}
	chain := &BlockChain{db: db, cacheConfig: &CacheConfig{HistoryBlocks: 3}}

	if err := chain.pruneBlockHistory(9); err != nil {
		t.Fatal(err)
	}
	if tail, _ := db.Tail(); tail != 0 {
		t.Fatalf("tail advanced without index markers: got %d, want 0", tail)
	}
	rawdb.WriteTxIndexTail(db, 7)
	if err := chain.pruneBlockHistory(9); err != nil {
		t.Fatal(err)
	}
	if tail, _ := db.Tail(); tail != 0 {
		t.Fatalf("tail advanced without reference index marker: got %d, want 0", tail)
	}
	rawdb.WriteReferenceIndexTail(db, 7)
	if err := chain.pruneBlockHistory(9); err != nil {
		t.Fatal(err)
	}
	if tail, _ := db.Tail(); tail != 7 {
		t.Fatalf("tail = %d, want 7", tail)
	}
	if ok, _ := db.HasAncient("headers", 6); ok {
		t.Fatal("block below history tail is still visible")
	}
	if ok, _ := db.HasAncient("headers", 7); !ok {
		t.Fatal("first retained block is not visible")
	}
}
