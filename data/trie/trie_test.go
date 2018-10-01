// Copyright 2014 The go-ethereum Authors
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
package trie_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie/encoding"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie/mockCrypto"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie/mockDatabase"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/ElrondNetwork/elrond-go-sandbox/data/trie/rlp"

	"github.com/davecgh/go-spew/spew"
)

func init() {
	spew.Config.Indent = "    "
	spew.Config.DisableMethods = false

	trie.SetDefaultHasher(&mockCrypto.Keccak256{})
}

// Used for testing
func newEmpty() *trie.Trie {
	tr, _ := trie.NewTrie(encoding.Hash{}, trie.NewDatabase(mockDatabase.NewMemDatabase()))
	return tr
}

func TestEmptyTrie(t *testing.T) {
	var tr trie.Trie
	res := tr.Hash()
	exp := trie.EmptyRoot
	if res != encoding.Hash(exp) {
		t.Errorf("expected %x got %x", exp, res)
	}
}

func TestNull(t *testing.T) {
	var tr trie.Trie
	key := make([]byte, 32)
	value := []byte("test")
	tr.Update(key, value)
	if !bytes.Equal(tr.Get(key), value) {
		t.Fatal("wrong value")
	}
}

func TestMissingRoot(t *testing.T) {
	tr, err := trie.NewTrie(encoding.HexToHash("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33"), trie.NewDatabase(mockDatabase.NewMemDatabase()))
	if tr != nil {
		t.Error("New returned non-nil trie for invalid root")
	}
	if _, ok := err.(*encoding.MissingNodeError); !ok {
		t.Errorf("New returned wrong error: %v", err)
	}
}

func TestMissingNodeDisk(t *testing.T)    { testMissingNode(t, false) }
func TestMissingNodeMemonly(t *testing.T) { testMissingNode(t, true) }

func testMissingNode(t *testing.T, memonly bool) {
	diskdb := mockDatabase.NewMemDatabase()
	triedb := trie.NewDatabase(diskdb)

	tr, _ := trie.NewTrie(encoding.Hash{}, triedb)
	updateString(tr, "120000", "qwerqwerqwerqwerqwerqwerqwerqwer")
	updateString(tr, "123456", "asdfasdfasdfasdfasdfasdfasdfasdf")
	root, _ := tr.Commit(nil)
	if !memonly {
		triedb.Commit(root, true)
	}

	tr, _ = trie.NewTrie(root, triedb)
	_, err := tr.TryGet([]byte("120000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	_, err = tr.TryGet([]byte("120099"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	_, err = tr.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	err = tr.TryUpdate([]byte("120099"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	err = tr.TryDelete([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	hash := encoding.HexToHash("0xe1d943cc8f061a0c0b98162830b970395ac9315654824bf21b73b891365262f9")
	if memonly {
		delete(triedb.NodesTest(), hash)
	} else {
		diskdb.Delete(hash[:])
	}

	tr, _ = trie.NewTrie(root, triedb)
	_, err = tr.TryGet([]byte("120000"))
	if _, ok := err.(*encoding.MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	_, err = tr.TryGet([]byte("120099"))
	if _, ok := err.(*encoding.MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	_, err = tr.TryGet([]byte("123456"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	err = tr.TryUpdate([]byte("120099"), []byte("zxcv"))
	if _, ok := err.(*encoding.MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
	tr, _ = trie.NewTrie(root, triedb)
	err = tr.TryDelete([]byte("123456"))
	if _, ok := err.(*encoding.MissingNodeError); !ok {
		t.Errorf("Wrong error: %v", err)
	}
}

func TestInsert(t *testing.T) {
	tr := newEmpty()

	updateString(tr, "doe", "reindeer")
	updateString(tr, "dog", "puppy")
	updateString(tr, "dogglesworth", "cat")

	exp := encoding.HexToHash("8aad789dff2f538bca5d8ea56e8abe10f4c7ba3a5dea95fea4cd6e7c3a1168d3")
	root := tr.Hash()
	if root != exp {
		t.Errorf("exp %x got %x", exp, root)
	}

	tr = newEmpty()
	updateString(tr, "A", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	exp = encoding.HexToHash("d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab")
	root, err := tr.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if root != exp {
		t.Errorf("exp %x got %x", exp, root)
	}
}

func TestGet(t *testing.T) {
	tr := newEmpty()
	updateString(tr, "doe", "reindeer")
	updateString(tr, "dog", "puppy")
	updateString(tr, "dogglesworth", "cat")

	for i := 0; i < 2; i++ {
		res := getString(tr, "dog")
		if !bytes.Equal(res, []byte("puppy")) {
			t.Errorf("expected puppy got %x", res)
		}

		unknown := getString(tr, "unknown")
		if unknown != nil {
			t.Errorf("expected nil got %x", unknown)
		}

		if i == 1 {
			return
		}
		tr.Commit(nil)
	}
}

func TestDelete(t *testing.T) {
	tr := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			updateString(tr, val.k, val.v)
		} else {
			deleteString(tr, val.k)
		}
	}

	hash := tr.Hash()
	exp := encoding.HexToHash("5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestEmptyValues(t *testing.T) {
	tr := newEmpty()

	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		updateString(tr, val.k, val.v)
	}

	hash := tr.Hash()
	exp := encoding.HexToHash("5991bb8c6514148a29db676a14ac506cd2cd5775ace63c30a4fe457715e9ac84")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestReplication(t *testing.T) {
	tr := newEmpty()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
	}
	for _, val := range vals {
		updateString(tr, val.k, val.v)
	}
	exp, err := tr.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}

	// create a new trie on top of the database and check that lookups work.
	trie2, err := trie.NewTrie(exp, tr.DB())
	if err != nil {
		t.Fatalf("can't recreate trie at %x: %v", exp, err)
	}
	for _, kv := range vals {
		if string(getString(trie2, kv.k)) != kv.v {
			t.Errorf("trie2 doesn't have %q => %q", kv.k, kv.v)
		}
	}
	hash, err := trie2.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}

	// perform some insertions on the new trie.
	vals2 := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		// {"shaman", "horse"},
		// {"doge", "coin"},
		// {"ether", ""},
		// {"dog", "puppy"},
		// {"somethingveryoddindeedthis is", "myothernodedata"},
		// {"shaman", ""},
	}
	for _, val := range vals2 {
		updateString(trie2, val.k, val.v)
	}
	if hash := trie2.Hash(); hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}
}

func TestLargeValue(t *testing.T) {
	tr := newEmpty()
	tr.Update([]byte("key1"), []byte{99, 99, 99, 99})
	tr.Update([]byte("key2"), bytes.Repeat([]byte{1}, 32))
	tr.Hash()
}

type countingDB struct {
	trie.PersistDB
	gets map[string]int
}

func (db *countingDB) Get(key []byte) ([]byte, error) {
	db.gets[string(key)]++
	return db.PersistDB.Get(key)
}

// TestCacheUnload checks that decoded nodes are unloaded after a
// certain number of commit operations.
func TestCacheUnload(t *testing.T) {
	// Create test trie with two branches.
	tr := newEmpty()
	key1 := "---------------------------------"
	key2 := "---some other branch"
	updateString(tr, key1, "this is the branch of key1.")
	updateString(tr, key2, "this is the branch of key2.")

	root, _ := tr.Commit(nil)
	tr.DB().Commit(root, true)

	// Commit the trie repeatedly and access key1.
	// The branch containing it is loaded from DB exactly two times:
	// in the 0th and 6th iteration.
	db := &countingDB{PersistDB: tr.DB().DiskDB(), gets: make(map[string]int)}
	tr, _ = trie.NewTrie(root, trie.NewDatabase(db))
	tr.SetCacheLimit(5)
	for i := 0; i < 12; i++ {
		getString(tr, key1)
		tr.Commit(nil)
	}
	// Check that it got loaded two times.
	for dbkey, count := range db.gets {
		if count != 2 {
			t.Errorf("db key %x loaded %d times, want %d times", []byte(dbkey), count, 2)
		}
	}
}

// randTest performs random trie operations.
// Instances of this test are created by Generate.
type randTest []randTestStep

type randTestStep struct {
	op    int
	key   []byte // for opUpdate, opDelete, opGet
	value []byte // for opUpdate
	err   error  // for debugging
}

const (
	opUpdate = iota
	opDelete
	opGet
	opCommit
	opHash
	opReset
	opItercheckhash
	opCheckCacheInvariant
	opMax // boundary value, not an actual op
)

func (randTest) Generate(r *rand.Rand, size int) reflect.Value {
	var allKeys [][]byte
	genKey := func() []byte {
		if len(allKeys) < 2 || r.Intn(100) < 10 {
			// new key
			key := make([]byte, r.Intn(50))
			r.Read(key)
			allKeys = append(allKeys, key)
			return key
		}
		// use existing key
		return allKeys[r.Intn(len(allKeys))]
	}

	var steps randTest
	for i := 0; i < size; i++ {
		step := randTestStep{op: r.Intn(opMax)}
		switch step.op {
		case opUpdate:
			step.key = genKey()
			step.value = make([]byte, 8)
			binary.BigEndian.PutUint64(step.value, uint64(i))
		case opGet, opDelete:
			step.key = genKey()
		}
		steps = append(steps, step)
	}
	return reflect.ValueOf(steps)
}

func runRandTest(rt randTest) bool {
	triedb := trie.NewDatabase(mockDatabase.NewMemDatabase())

	tr, _ := trie.NewTrie(encoding.Hash{}, triedb)
	values := make(map[string]string) // tracks content of the trie

	for i, step := range rt {
		switch step.op {
		case opUpdate:
			tr.Update(step.key, step.value)
			values[string(step.key)] = string(step.value)
		case opDelete:
			tr.Delete(step.key)
			delete(values, string(step.key))
		case opGet:
			v := tr.Get(step.key)
			want := values[string(step.key)]
			if string(v) != want {
				rt[i].err = fmt.Errorf("mismatch for key 0x%x, got 0x%x want 0x%x", step.key, v, want)
			}
		case opCommit:
			_, rt[i].err = tr.Commit(nil)
		case opHash:
			tr.Hash()
		case opReset:
			hash, err := tr.Commit(nil)
			if err != nil {
				rt[i].err = err
				return false
			}
			newtr, err := trie.NewTrie(hash, triedb)
			if err != nil {
				rt[i].err = err
				return false
			}
			tr = newtr
		case opItercheckhash:
			checktr, _ := trie.NewTrie(encoding.Hash{}, triedb)
			it := trie.NewIterator(tr.NodeIterator(nil))
			for it.Next() {
				checktr.Update(it.Key, it.Value)
			}
			if tr.Hash() != checktr.Hash() {
				rt[i].err = fmt.Errorf("hash mismatch in opItercheckhash")
			}
		case opCheckCacheInvariant:
			rt[i].err = checkCacheInvariant(tr.RootNode(), nil, tr.Cachegen(), false, 0)
		}
		// Abort the test on error.
		if rt[i].err != nil {
			return false
		}
	}
	return true
}

func checkCacheInvariant(n, parent trie.Node, parentCachegen uint16, parentDirty bool, depth int) error {
	var children []trie.Node
	var flag trie.NodeFlag
	switch n := n.(type) {
	case *trie.ShortNode:
		flag = n.Flags()
		children = []trie.Node{n.Val}
	case *trie.FullNode:
		flag = n.Flags()
		children = n.ChildrenNodes()
	default:
		return nil
	}

	errorf := func(format string, args ...interface{}) error {
		msg := fmt.Sprintf(format, args...)
		msg += fmt.Sprintf("\nat depth %d node %s", depth, spew.Sdump(n))
		msg += fmt.Sprintf("parent: %s", spew.Sdump(parent))
		return errors.New(msg)
	}
	if flag.Gen() > parentCachegen {
		return errorf("cache invariant violation: %d > %d\n", flag.Gen(), parentCachegen)
	}
	if depth > 0 && !parentDirty && flag.Dirty() {
		return errorf("cache invariant violation: %d > %d\n", flag.Gen(), parentCachegen)
	}
	for _, child := range children {
		if err := checkCacheInvariant(child, n, flag.Gen(), flag.Dirty(), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func TestRandom(t *testing.T) {
	if err := quick.Check(runRandTest, nil); err != nil {
		if cerr, ok := err.(*quick.CheckError); ok {
			t.Fatalf("random test iteration %d failed: %s", cerr.Count, spew.Sdump(cerr.In))
		}
		t.Fatal(err)
	}
}

func BenchmarkGet(b *testing.B)      { benchGet(b, false) }
func BenchmarkGetDB(b *testing.B)    { benchGet(b, true) }
func BenchmarkUpdateBE(b *testing.B) { benchUpdate(b, binary.BigEndian) }
func BenchmarkUpdateLE(b *testing.B) { benchUpdate(b, binary.LittleEndian) }

const benchElemCount = 20000

func benchGet(b *testing.B, commit bool) {
	tr := new(trie.Trie)
	if commit {
		_, tmpdb := tempDB()
		tr, _ = trie.NewTrie(encoding.Hash{}, tmpdb)
	}
	k := make([]byte, 32)
	for i := 0; i < benchElemCount; i++ {
		binary.LittleEndian.PutUint64(k, uint64(i))
		tr.Update(k, k)
	}
	binary.LittleEndian.PutUint64(k, benchElemCount/2)
	if commit {
		tr.Commit(nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Get(k)
	}
	b.StopTimer()

	if commit {
		//ldb := trie.db.diskdb.(*ethdb.LDBDatabase)
		//ldb.Close()
		//os.RemoveAll(ldb.Path())
	}
}

func benchUpdate(b *testing.B, e binary.ByteOrder) *trie.Trie {
	tr := newEmpty()
	k := make([]byte, 32)
	for i := 0; i < b.N; i++ {
		e.PutUint64(k, uint64(i))
		tr.Update(k, k)
	}
	return tr
}

// Benchmarks the trie hashing. Since the trie caches the result of any operation,
// we cannot use b.N as the number of hashing rouns, since all rounds apart from
// the first one will be NOOP. As such, we'll use b.N as the number of account to
// insert into the trie before measuring the hashing.
func BenchmarkHash(b *testing.B) {
	// Make the random benchmark deterministic
	random := rand.New(rand.NewSource(0))

	// Create a realistic account trie to hash
	addresses := make([][20]byte, b.N)
	for i := 0; i < len(addresses); i++ {
		for j := 0; j < len(addresses[i]); j++ {
			addresses[i][j] = byte(random.Intn(256))
		}
	}
	accounts := make([][]byte, len(addresses))
	for i := 0; i < len(accounts); i++ {
		var (
			nonce   = uint64(random.Int63())
			balance = new(big.Int).Rand(random, new(big.Int).Exp(encoding.Big2, encoding.Big256, nil))
			root    = trie.EmptyRoot
			code    = trie.DefaultHasher().Compute("")
		)
		accounts[i], _ = rlp.EncodeToBytes([]interface{}{nonce, balance, root, code})
	}
	// Insert the accounts into the trie and hash it
	tr := newEmpty()
	for i := 0; i < len(addresses); i++ {
		tr.Update(trie.DefaultHasher().Compute(string(addresses[i][:])), accounts[i])
	}
	b.ResetTimer()
	b.ReportAllocs()
	tr.Hash()
}

func tempDB() (string, *trie.Database) {
	//dir, err := ioutil.TempDir("", "trie-bench")
	//if err != nil {
	//	panic(fmt.Sprintf("can't create temporary directory: %v", err))
	//}
	//diskdb, err := ethdb.NewLDBDatabase(dir, 256, 0)
	diskdb := mockDatabase.NewMemDatabase()
	//if err != nil {
	//	panic(fmt.Sprintf("can't create temporary database: %v", err))
	//}
	return "MEM", trie.NewDatabase(diskdb)
}

func getString(trie *trie.Trie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateString(trie *trie.Trie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteString(trie *trie.Trie, k string) {
	trie.Delete([]byte(k))
}
