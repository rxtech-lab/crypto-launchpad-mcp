package e2e

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContractDeploymentHelpers tests the new deployment helper functions
func TestContractDeploymentHelpers(t *testing.T) {
	// Skip this test if anvil is not running
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	err := setup.VerifyEthereumConnection()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
		return
	}

	t.Run("DeploySimpleERC20WithHelpers", func(t *testing.T) {
		account := setup.GetPrimaryTestAccount()

		// Deploy using helper function
		result, err := setup.DeployContract(
			account,
			GetSimpleERC20Contract(),
			"SimpleERC20",
			"HelperToken",       // name
			"HELP",              // symbol
			big.NewInt(1000000), // totalSupply
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		t.Logf("✓ Contract deployed at: %s", result.ContractAddress.Hex())
		t.Logf("✓ Transaction hash: %s", result.TransactionHash.Hex())
		t.Logf("✓ Gas used: %d", result.Receipt.GasUsed)

		// Test view functions
		nameResult, err := setup.CallContractView(result, "name")
		require.NoError(t, err)
		require.Len(t, nameResult, 1)
		assert.Equal(t, "HelperToken", nameResult[0].(string))

		symbolResult, err := setup.CallContractView(result, "symbol")
		require.NoError(t, err)
		require.Len(t, symbolResult, 1)
		assert.Equal(t, "HELP", symbolResult[0].(string))

		// Test balance
		balanceResult, err := setup.CallContractView(result, "balanceOf", account.Address)
		require.NoError(t, err)
		require.Len(t, balanceResult, 1)

		expectedBalance := new(big.Int).Mul(big.NewInt(1000000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
		assert.Equal(t, expectedBalance, balanceResult[0].(*big.Int))

		t.Logf("✓ Name: %s", nameResult[0].(string))
		t.Logf("✓ Symbol: %s", symbolResult[0].(string))
		t.Logf("✓ Deployer balance: %s", balanceResult[0].(*big.Int).String())
	})

	t.Run("DeployMintableTokenWithHelpers", func(t *testing.T) {
		account := setup.GetSecondaryTestAccount()

		// Deploy using helper function
		result, err := setup.DeployContract(
			account,
			GetMintableTokenContract(),
			"MintableToken",
			"MintHelper",       // name
			"MHELP",            // symbol
			big.NewInt(100000), // initialSupply
		)
		require.NoError(t, err)
		require.NotNil(t, result)

		t.Logf("✓ Mintable contract deployed at: %s", result.ContractAddress.Hex())

		// Test owner
		ownerResult, err := setup.CallContractView(result, "owner")
		require.NoError(t, err)
		require.Len(t, ownerResult, 1)
		assert.Equal(t, account.Address, ownerResult[0])

		// Test minting functionality
		mintAmount := big.NewInt(5000)
		receipt, err := setup.CallContractTx(account, result, "mint", account.Address, mintAmount)
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status)

		t.Logf("✓ Minted %s tokens successfully", mintAmount.String())
		t.Logf("✓ Mint transaction gas used: %d", receipt.GasUsed)

		// Verify new balance
		balanceResult, err := setup.CallContractView(result, "balanceOf", account.Address)
		require.NoError(t, err)
		require.Len(t, balanceResult, 1)

		initialSupply := new(big.Int).Mul(big.NewInt(100000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
		expectedBalance := new(big.Int).Add(initialSupply, mintAmount)
		assert.Equal(t, expectedBalance, balanceResult[0].(*big.Int))

		t.Logf("✓ Final balance after mint: %s", balanceResult[0].(*big.Int).String())
	})
}
