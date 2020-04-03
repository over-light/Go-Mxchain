package upgrades

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ElrondNetwork/elrond-go/integrationTests/vm/arwen"
	"github.com/stretchr/testify/require"
)

func TestUpgrades_Hello(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.Close()

	fmt.Println("Deploy v1")

	context.ScCodeMetadata.Upgradeable = true
	err := context.DeploySC("../testdata/hello-v1/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(24), context.QuerySCInt("getUltimateAnswer", [][]byte{}))

	fmt.Println("Upgrade to v2")

	err = context.UpgradeSC("../testdata/hello-v2/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(42), context.QuerySCInt("getUltimateAnswer", [][]byte{}))

	fmt.Println("Upgrade to v3")

	err = context.UpgradeSC("../testdata/hello-v3/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, "forty-two", context.QuerySCString("getUltimateAnswer", [][]byte{}))
}

func TestUpgrades_HelloDoesNotUpgradeWhenNotUpgradeable(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.Close()

	fmt.Println("Deploy v1")

	context.ScCodeMetadata.Upgradeable = false
	err := context.DeploySC("../testdata/hello-v1/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(24), context.QuerySCInt("getUltimateAnswer", [][]byte{}))

	fmt.Println("Upgrade to v2 will not be performed")

	err = context.UpgradeSC("../testdata/hello-v2/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(24), context.QuerySCInt("getUltimateAnswer", [][]byte{}))
}

func TestUpgrades_HelloUpgradesToNotUpgradeable(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.Close()

	fmt.Println("Deploy v1")

	context.ScCodeMetadata.Upgradeable = true
	err := context.DeploySC("../testdata/hello-v1/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(24), context.QuerySCInt("getUltimateAnswer", [][]byte{}))

	fmt.Println("Upgrade to v2, becomes not upgradeable")

	context.ScCodeMetadata.Upgradeable = false
	err = context.UpgradeSC("../testdata/hello-v2/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(42), context.QuerySCInt("getUltimateAnswer", [][]byte{}))

	fmt.Println("Upgrade to v3")

	err = context.UpgradeSC("../testdata/hello-v3/answer.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(42), context.QuerySCInt("getUltimateAnswer", [][]byte{}))
}

func TestUpgrades_ParentAndChildContracts(t *testing.T) {
	context := arwen.SetupTestContext(t)
	defer context.Close()

	var parentAddress []byte
	var childAddress []byte
	owner := &context.Owner

	fmt.Println("Deploy parent")

	err := context.DeploySC("../testdata/upgrades-parent/parent.wasm", "")
	require.Nil(t, err)
	require.Equal(t, uint64(45), context.QuerySCInt("getUltimateAnswer", [][]byte{}))
	parentAddress = context.ScAddress

	fmt.Println("Deploy child v1")

	childInitialCode := arwen.GetSCCode("../testdata/hello-v1/answer.wasm")
	err = context.ExecuteSC(owner, "createChild@"+childInitialCode)
	require.Nil(t, err)

	fmt.Println("Aquire child address, do query")

	childAddress = context.QuerySCBytes("getChildAddress", [][]byte{})
	context.ScAddress = childAddress
	require.Equal(t, uint64(24), context.QuerySCInt("getUltimateAnswer", [][]byte{}))

	fmt.Println("Deploy child v2")
	context.ScAddress = parentAddress
	// We need to double hex-encode the code (so that we don't have to hex-encode in the contract).
	childUpgradedCode := arwen.GetSCCode("../testdata/hello-v2/answer.wasm")
	childUpgradedCode = hex.EncodeToString([]byte(childUpgradedCode))
	// Not supported at this moment.
	err = context.ExecuteSC(owner, "upgradeChild@"+childUpgradedCode)
	require.NotNil(t, err)
}
