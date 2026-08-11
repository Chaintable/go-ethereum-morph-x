// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package rawdb

import (
	"math/big"
	"testing"

	"github.com/morph-l2/go-ethereum/core/types"
	"github.com/morph-l2/go-ethereum/ethdb"
)

func makeReferenceIteratorChain(t *testing.T) (ethdb.Database, uint64) {
	t.Helper()

	db := NewMemoryDatabase()
	var block *types.Block
	for i := uint64(0); i <= 10; i++ {
		block = types.NewBlock(&types.Header{
			Number:     new(big.Int).SetUint64(i),
			Difficulty: big.NewInt(1),
			Time:       i,
		}, nil, nil, nil, newHasher())
		WriteBlock(db, block)
		WriteCanonicalHash(db, block.Hash(), i)
	}
	IndexReferences(db, 0, 11, nil)
	return db, 5
}

func TestUnindexReferencesStopsAtMissingBody(t *testing.T) {
	db, missing := makeReferenceIteratorChain(t)
	hash := ReadCanonicalHash(db, missing)
	DeleteBody(db, hash, missing)

	UnindexReferences(db, 0, 11, nil)
	tail := ReadReferenceIndexTail(db)
	if tail == nil || *tail != missing {
		t.Fatalf("reference index tail after missing body = %v, want %d", tail, missing)
	}
}

func TestUnindexReferencesStopsAtCorruptBody(t *testing.T) {
	db, corrupt := makeReferenceIteratorChain(t)
	hash := ReadCanonicalHash(db, corrupt)
	WriteBodyRLP(db, hash, corrupt, []byte{0xff})

	UnindexReferences(db, 0, 11, nil)
	tail := ReadReferenceIndexTail(db)
	if tail == nil || *tail != corrupt {
		t.Fatalf("reference index tail after corrupt body = %v, want %d", tail, corrupt)
	}
}
