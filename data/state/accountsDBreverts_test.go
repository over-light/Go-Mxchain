package state_test

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-sandbox/data/state"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/state/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func adbrCreateAccountsDB() *state.AccountsDB {
	marsh := mock.MarshalizerMock{}
	adb := state.NewAccountsDB(mock.NewMockTrie(), mock.HasherMock{}, &marsh)

	return adb
}

func adbrCreateAddress(t testing.TB, buff []byte) *state.Address {
	adr, err := state.FromPubKeyBytes(buff, mock.HasherMock{})
	assert.Nil(t, err)

	return adr
}

func adbrEmulateBalanceTxExecution(acntSrc, acntDest *state.AccountState,
	handler state.AccountsHandler, value *big.Int) error {
	srcVal := acntSrc.Balance()
	destVal := acntDest.Balance()

	if srcVal.Cmp(value) < 0 {
		return errors.New("not enough funds")
	}

	//substract value from src
	err := acntSrc.SetBalance(handler, srcVal.Sub(&srcVal, value))
	if err != nil {
		return err
	}

	//add value to dest
	err = acntDest.SetBalance(handler, destVal.Add(&destVal, value))
	if err != nil {
		return err
	}

	//increment src's nonce
	err = acntSrc.SetNonce(handler, acntSrc.Nonce()+1)
	if err != nil {
		return err
	}

	return nil
}

func adbrEmulateBalanceTxSafeExecution(acntSrc, acntDest *state.AccountState,
	handler state.AccountsHandler, value *big.Int) {
	snapshot := handler.Journal().Len()

	err := adbrEmulateBalanceTxExecution(acntSrc, acntDest, handler, value)

	if err != nil {
		fmt.Printf("!!!! Error executing tx (value: %v), reverting...\n", value)
		err = handler.Journal().RevertFromSnapshot(snapshot)

		if err != nil {
			panic(err)
		}
	}
}

func adbrPrintAccount(as *state.AccountState, tag string) {
	bal := as.Balance()
	fmt.Printf("%s address: %s\n", tag, base64.StdEncoding.EncodeToString(as.Addr.Bytes()))
	fmt.Printf("     Nonce: %d\n", as.Nonce())
	fmt.Printf("     Balance: %d\n", bal.Uint64())
	fmt.Printf("     Code hash: %v\n", base64.StdEncoding.EncodeToString(as.CodeHash()))
	fmt.Printf("     Root: %v\n\n", base64.StdEncoding.EncodeToString(as.Root()))
}

func TestAccountsDB_RevertNonceStepByStep_AccountData_ShouldWork(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	testHash1 := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHash2 := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adr1 := adbrCreateAddress(t, testHash1)
	adr2 := adbrCreateAddress(t, testHash2)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()
	hrEmpty := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - empty: %v\n", hrEmpty)

	//Step 2. create 2 new accounts
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	snapshotCreated1 := adb.Journal().Len()
	hrCreated1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	fmt.Printf("State root - created 1-st account: %v\n", hrCreated1)

	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	snapshotCreated2 := adb.Journal().Len()
	hrCreated2 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	fmt.Printf("State root - created 2-nd account: %v\n", hrCreated2)

	//Test 2.1. test that hashes and snapshots ID are different
	assert.NotEqual(t, snapshotCreated2, snapshotCreated1)
	assert.NotEqual(t, hrCreated1, hrCreated2)

	//Save the preset snapshot id
	snapshotPreSet := adb.Journal().Len()

	//Step 3. Set Nonces and save data
	err = state1.SetNonce(adb, 40)
	assert.Nil(t, err)
	hrWithNonce1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - account with nonce 40: %v\n", hrWithNonce1)

	err = state2.SetNonce(adb, 50)
	assert.Nil(t, err)
	hrWithNonce2 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - account with nonce 50: %v\n", hrWithNonce2)

	//Test 3.1. current root hash shall not match created root hash hrCreated2
	assert.NotEqual(t, hrCreated2, adb.MainTrie.Root())

	//Step 4. Revert account nonce and test
	err = adb.Journal().RevertFromSnapshot(snapshotPreSet)
	assert.Nil(t, err)

	//Test 4.1. current root hash shall match created root hash hrCreated
	hrFinal := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	assert.Equal(t, hrCreated2, hrFinal)
	fmt.Printf("State root - reverted last 2 nonces set: %v\n", hrFinal)
}

func TestAccountsDB_RevertBalanceStepByStep_AccountData_ShouldWork(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	testHash1 := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHash2 := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adr1 := adbrCreateAddress(t, testHash1)
	adr2 := adbrCreateAddress(t, testHash2)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()
	hrEmpty := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - empty: %v\n", hrEmpty)

	//Step 2. create 2 new accounts
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	snapshotCreated1 := adb.Journal().Len()
	hrCreated1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	fmt.Printf("State root - created 1-st account: %v\n", hrCreated1)

	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	snapshotCreated2 := adb.Journal().Len()
	hrCreated2 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	fmt.Printf("State root - created 2-nd account: %v\n", hrCreated2)

	//Test 2.1. test that hashes and snapshots ID are different
	assert.NotEqual(t, snapshotCreated2, snapshotCreated1)
	assert.NotEqual(t, hrCreated1, hrCreated2)

	//Save the preset snapshot id
	snapshotPreSet := adb.Journal().Len()

	//Step 3. Set balances and save data
	err = state1.SetBalance(adb, big.NewInt(40))
	assert.Nil(t, err)
	hrWithBalance1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - account with balance 40: %v\n", hrWithBalance1)

	err = state2.SetBalance(adb, big.NewInt(50))
	assert.Nil(t, err)
	hrWithBalance2 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - account with balance 50: %v\n", hrWithBalance2)

	//Test 3.1. current root hash shall not match created root hash hrCreated2
	assert.NotEqual(t, hrCreated2, adb.MainTrie.Root())

	//Step 4. Revert account balances and test
	err = adb.Journal().RevertFromSnapshot(snapshotPreSet)
	assert.Nil(t, err)

	//Test 4.1. current root hash shall match created root hash hrCreated
	hrFinal := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	assert.Equal(t, hrCreated2, hrFinal)
	fmt.Printf("State root - reverted last 2 balance set: %v\n", hrFinal)
}

func TestAccountsDB_RevertCodeStepByStep_AccountData_ShouldWork(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	//adr1 puts code hash + code inside trie. adr2 has the same code hash
	//revert should work

	testHash1 := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHash2 := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adr1 := adbrCreateAddress(t, testHash1)
	adr2 := adbrCreateAddress(t, testHash2)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()
	hrEmpty := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - empty: %v\n", hrEmpty)

	//Step 2. create 2 new accounts
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	err = adb.PutCode(state1, []byte{65, 66, 67})
	assert.Nil(t, err)
	snapshotCreated1 := adb.Journal().Len()
	hrCreated1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	fmt.Printf("State root - created 1-st account: %v\n", hrCreated1)

	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	err = adb.PutCode(state2, []byte{65, 66, 67})
	assert.Nil(t, err)
	snapshotCreated2 := adb.Journal().Len()
	hrCreated2 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	fmt.Printf("State root - created 2-nd account: %v\n", hrCreated2)

	//Test 2.1. test that hashes and snapshots ID are different
	assert.NotEqual(t, snapshotCreated2, snapshotCreated1)
	assert.NotEqual(t, hrCreated1, hrCreated2)

	//Step 3. Revert second account
	err = adb.Journal().RevertFromSnapshot(snapshotCreated1)
	assert.Nil(t, err)

	//Test 3.1. current root hash shall match created root hash hrCreated1
	hrCrt := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	assert.Equal(t, hrCreated1, hrCrt)
	fmt.Printf("State root - reverted last account: %v\n", hrCrt)

	//Step 4. Revert first account
	err = adb.Journal().RevertFromSnapshot(0)
	assert.Nil(t, err)

	//Test 4.1. current root hash shall match empty root hash
	hrCrt = base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	assert.Equal(t, hrEmpty, hrCrt)
	fmt.Printf("State root - reverted first account: %v\n", hrCrt)
}

func TestAccountsDB_RevertDataStepByStep_AccountData_ShouldWork(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	//adr1 puts data inside trie. adr2 puts the same data
	//revert should work

	testHash1 := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHash2 := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adr1 := adbrCreateAddress(t, testHash1)
	adr2 := adbrCreateAddress(t, testHash2)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()
	hrEmpty := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - empty: %v\n", hrEmpty)

	//Step 2. create 2 new accounts
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	state1.SaveKeyValue([]byte{65, 66, 67}, []byte{32, 33, 34})
	err = adb.SaveData(state1)
	assert.Nil(t, err)
	snapshotCreated1 := adb.Journal().Len()
	hrCreated1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	hrRoot1 := base64.StdEncoding.EncodeToString(state1.DataTrie.Root())

	fmt.Printf("State root - created 1-st account: %v\n", hrCreated1)
	fmt.Printf("Data root - 1-st account: %v\n", hrRoot1)

	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	state2.SaveKeyValue([]byte{65, 66, 67}, []byte{32, 33, 34})
	err = adb.SaveData(state2)
	assert.Nil(t, err)
	snapshotCreated2 := adb.Journal().Len()
	hrCreated2 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	hrRoot2 := base64.StdEncoding.EncodeToString(state1.DataTrie.Root())

	fmt.Printf("State root - created 2-nd account: %v\n", hrCreated2)
	fmt.Printf("Data root - 2-nd account: %v\n", hrRoot2)

	//Test 2.1. test that hashes and snapshots ID are different
	assert.NotEqual(t, snapshotCreated2, snapshotCreated1)
	assert.NotEqual(t, hrCreated1, hrCreated2)

	//Test 2.2 test whether the datatrie roots match
	assert.Equal(t, hrRoot1, hrRoot2)

	//Step 3. Revert 2-nd account ant test roots
	err = adb.Journal().RevertFromSnapshot(snapshotCreated1)
	assert.Nil(t, err)
	hrCreated2Rev := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	assert.Equal(t, hrCreated1, hrCreated2Rev)

	//Step 4. Revert 1-st account ant test roots
	err = adb.Journal().RevertFromSnapshot(0)
	assert.Nil(t, err)
	hrCreated1Rev := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())

	assert.Equal(t, hrEmpty, hrCreated1Rev)
}

func TestAccountsDB_RevertDataStepByStepWithCommits_AccountData_ShouldWork(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	//adr1 puts data inside trie. adr2 puts the same data
	//revert should work

	testHash1 := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHash2 := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adr1 := adbrCreateAddress(t, testHash1)
	adr2 := adbrCreateAddress(t, testHash2)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()
	hrEmpty := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("State root - empty: %v\n", hrEmpty)

	//Step 2. create 2 new accounts
	state1, err := adb.GetOrCreateAccount(*adr1)
	assert.Nil(t, err)
	state1.SaveKeyValue([]byte{65, 66, 67}, []byte{32, 33, 34})
	err = adb.SaveData(state1)
	assert.Nil(t, err)
	snapshotCreated1 := adb.Journal().Len()
	hrCreated1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	hrRoot1 := base64.StdEncoding.EncodeToString(state1.DataTrie.Root())

	fmt.Printf("State root - created 1-st account: %v\n", hrCreated1)
	fmt.Printf("Data root - 1-st account: %v\n", hrRoot1)

	state2, err := adb.GetOrCreateAccount(*adr2)
	assert.Nil(t, err)
	state2.SaveKeyValue([]byte{65, 66, 67}, []byte{32, 33, 34})
	err = adb.SaveData(state2)
	assert.Nil(t, err)
	snapshotCreated2 := adb.Journal().Len()
	hrCreated2 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	hrRoot2 := base64.StdEncoding.EncodeToString(state1.DataTrie.Root())

	fmt.Printf("State root - created 2-nd account: %v\n", hrCreated2)
	fmt.Printf("Data root - 2-nd account: %v\n", hrRoot2)

	//Test 2.1. test that hashes and snapshots ID are different
	assert.NotEqual(t, snapshotCreated2, snapshotCreated1)
	assert.NotEqual(t, hrCreated1, hrCreated2)

	//Test 2.2 test whether the datatrie roots match
	assert.Equal(t, hrRoot1, hrRoot2)

	//Step 3. Commit
	rootCommit, err := adb.Commit()
	hrCommit := base64.StdEncoding.EncodeToString(rootCommit)
	fmt.Printf("State root - committed: %v\n", hrCommit)

	//Step 4. 2-nd account changes its data
	snapshotMod := adb.Journal().Len()
	state2.SaveKeyValue([]byte{65, 66, 67}, []byte{32, 33, 35})
	err = adb.SaveData(state2)
	assert.Nil(t, err)
	hrCreated2p1 := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	hrRoot2p1 := base64.StdEncoding.EncodeToString(state2.DataTrie.Root())

	fmt.Printf("State root - modified 2-nd account: %v\n", hrCreated2p1)
	fmt.Printf("Data root - 2-nd account: %v\n", hrRoot2p1)

	//Test 4.1 test that hashes are different
	assert.NotEqual(t, hrCreated2p1, hrCreated2)

	//Test 4.2 test whether the datatrie roots match/mismatch
	assert.Equal(t, hrRoot1, hrRoot2)
	assert.NotEqual(t, hrRoot2, hrRoot2p1)

	//Step 5. Revert 2-nd account modification
	err = adb.Journal().RevertFromSnapshot(snapshotMod)
	assert.Nil(t, err)
	hrCreated2Rev := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	hrRoot2Rev := base64.StdEncoding.EncodeToString(state2.DataTrie.Root())
	fmt.Printf("State root - reverted 2-nd account: %v\n", hrCreated2Rev)
	fmt.Printf("Data root - 2-nd account: %v\n", hrRoot2Rev)
	assert.Equal(t, hrCommit, hrCreated2Rev)
	assert.Equal(t, hrRoot2, hrRoot2Rev)
}

func TestAccountsDB_ExecBalanceTxExecution(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	testHashSrc := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHashDest := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adrSrc := adbrCreateAddress(t, testHashSrc)
	adrDest := adbrCreateAddress(t, testHashDest)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()

	acntSrc, err := adb.GetOrCreateAccount(*adrSrc)
	assert.Nil(t, err)
	acntDest, err := adb.GetOrCreateAccount(*adrDest)
	assert.Nil(t, err)

	//Set a high balance to src's account
	err = acntSrc.SetBalance(adb, big.NewInt(1000))
	assert.Nil(t, err)

	hrOriginal := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("Original root hash: %s\n", hrOriginal)

	adbrPrintAccount(acntSrc, "Source")
	adbrPrintAccount(acntDest, "Destination")

	fmt.Println("Executing OK transaction...")
	adbrEmulateBalanceTxSafeExecution(acntSrc, acntDest, adb, big.NewInt(64))

	hrOK := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("After executing an OK tx root hash: %s\n", hrOK)

	adbrPrintAccount(acntSrc, "Source")
	adbrPrintAccount(acntDest, "Destination")

	fmt.Println("Executing NOK transaction...")
	adbrEmulateBalanceTxSafeExecution(acntSrc, acntDest, adb, big.NewInt(10000))

	hrNok := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("After executing a NOK tx root hash: %s\n", hrNok)

	adbrPrintAccount(acntSrc, "Source")
	adbrPrintAccount(acntDest, "Destination")

	assert.NotEqual(t, hrOriginal, hrOK)
	assert.Equal(t, hrOK, hrNok)

}

func TestAccountsDB_ExecALotOfBalanceTxOK(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	testHashSrc := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHashDest := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adrSrc := adbrCreateAddress(t, testHashSrc)
	adrDest := adbrCreateAddress(t, testHashDest)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()

	acntSrc, err := adb.GetOrCreateAccount(*adrSrc)
	assert.Nil(t, err)
	acntDest, err := adb.GetOrCreateAccount(*adrDest)
	assert.Nil(t, err)

	//Set a high balance to src's account
	err = acntSrc.SetBalance(adb, big.NewInt(10000000))
	assert.Nil(t, err)

	hrOriginal := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("Original root hash: %s\n", hrOriginal)

	for i := 1; i <= 1000; i++ {
		err := adbrEmulateBalanceTxExecution(acntSrc, acntDest, adb, big.NewInt(int64(i)))

		assert.Nil(t, err)
	}

	adbrPrintAccount(acntSrc, "Source")
	adbrPrintAccount(acntDest, "Destination")
}

func TestAccountsDB_ExecALotOfBalanceTxOKorNOK(t *testing.T) {
	t.Parallel()

	hasher := mock.HasherMock{}

	testHashSrc := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHashDest := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adrSrc := adbrCreateAddress(t, testHashSrc)
	adrDest := adbrCreateAddress(t, testHashDest)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()

	acntSrc, err := adb.GetOrCreateAccount(*adrSrc)
	assert.Nil(t, err)
	acntDest, err := adb.GetOrCreateAccount(*adrDest)
	assert.Nil(t, err)

	//Set a high balance to src's account
	err = acntSrc.SetBalance(adb, big.NewInt(10000000))
	assert.Nil(t, err)

	hrOriginal := base64.StdEncoding.EncodeToString(adb.MainTrie.Root())
	fmt.Printf("Original root hash: %s\n", hrOriginal)

	st := time.Now()
	for i := 1; i <= 1000; i++ {
		err := adbrEmulateBalanceTxExecution(acntSrc, acntDest, adb, big.NewInt(int64(i)))

		assert.Nil(t, err)

		err = adbrEmulateBalanceTxExecution(acntDest, acntSrc, adb, big.NewInt(int64(1000000)))

		assert.NotNil(t, err)
	}

	fmt.Printf("Done in %v\n", time.Now().Sub(st))

	adbrPrintAccount(acntSrc, "Source")
	adbrPrintAccount(acntDest, "Destination")
}

func BenchmarkTxExecution(b *testing.B) {
	hasher := mock.HasherMock{}

	testHashSrc := hasher.Compute("ABCDEFGHIJKLMNOP")
	testHashDest := hasher.Compute("ABCDEFGHIJKLMNOPQ")
	adrSrc := adbrCreateAddress(b, testHashSrc)
	adrDest := adbrCreateAddress(b, testHashDest)

	//Step 1. create accounts objects
	adb := adbrCreateAccountsDB()

	acntSrc, err := adb.GetOrCreateAccount(*adrSrc)
	assert.Nil(b, err)
	acntDest, err := adb.GetOrCreateAccount(*adrDest)
	assert.Nil(b, err)

	//Set a high balance to src's account
	err = acntSrc.SetBalance(adb, big.NewInt(10000000))
	assert.Nil(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		adbrEmulateBalanceTxSafeExecution(acntSrc, acntDest, adb, big.NewInt(1))
	}
}
