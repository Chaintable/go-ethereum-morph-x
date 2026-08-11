// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"math/big"
	"strings"
	"testing"

	"github.com/morph-l2/go-ethereum/common"
	"github.com/morph-l2/go-ethereum/core/types"
	"github.com/morph-l2/go-ethereum/ethdb/memorydb"
)

func TestDatabaseReopensAfterAncientTailDeletion(t *testing.T) {
	dir := t.TempDir()
	kv := memorydb.New()
	db, err := NewDatabaseWithFreezer(kv, dir, "", false)
	if err != nil {
		t.Fatal(err)
	}

	blocks := make([]*types.Block, 0, 10)
	parent := common.Hash{}
	for i := uint64(0); i < 10; i++ {
		block := types.NewBlockWithHeader(&types.Header{
			ParentHash:  parent,
			Number:      new(big.Int).SetUint64(i),
			Difficulty:  big.NewInt(1),
			UncleHash:   types.EmptyUncleHash,
			TxHash:      types.EmptyRootHash,
			ReceiptHash: types.EmptyRootHash,
		})
		blocks = append(blocks, block)
		parent = block.Hash()
	}
	if _, err := WriteAncientBlocks(db, blocks, make([]types.Receipts, len(blocks)), big.NewInt(1)); err != nil {
		t.Fatal(err)
	}
	WriteBlock(db, blocks[0])
	WriteCanonicalHash(db, blocks[0].Hash(), 0)

	hot := types.NewBlockWithHeader(&types.Header{
		ParentHash:  blocks[len(blocks)-1].Hash(),
		Number:      big.NewInt(10),
		Difficulty:  big.NewInt(1),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
	})
	WriteBlock(db, hot)
	WriteCanonicalHash(db, hot.Hash(), hot.NumberU64())
	WriteHeadHeaderHash(db, hot.Hash())
	WriteHeadBlockHash(db, hot.Hash())
	WriteHeadFastBlockHash(db, hot.Hash())

	if err := db.TruncateTail(7); err != nil {
		t.Fatal(err)
	}
	if err := db.Sync(); err != nil {
		t.Fatal(err)
	}
	first := db.(*freezerdb)
	if err := first.AncientStore.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewDatabaseWithFreezer(kv, dir, "", false)
	if err != nil {
		t.Fatalf("failed to reopen database after tail deletion: %v", err)
	}
	defer reopened.Close()
	if tail, err := reopened.Tail(); err != nil || tail != 7 {
		t.Fatalf("reopened tail = %d, %v; want 7", tail, err)
	}

	bad := types.NewBlockWithHeader(&types.Header{
		Number:      big.NewInt(10),
		Difficulty:  big.NewInt(1),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
	})
	WriteBlock(reopened, bad)
	WriteCanonicalHash(reopened, bad.Hash(), bad.NumberU64())
	freezer := reopened.(*freezerdb).AncientStore.(*freezer)
	if err := validateFreezerBoundary(kv, freezer, 10); err == nil || !strings.Contains(err.Error(), "chain boundary mismatch") {
		t.Fatalf("boundary validation error = %v, want chain boundary mismatch", err)
	}
}
