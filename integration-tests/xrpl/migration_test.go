//go:build integrationtests

package xrpl

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/CoreumFoundation/coreum/v5/testutil/integration"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
)

func TestContractMigration(t *testing.T) {
	// since we don't add state specific changes we test contract migration with the same source code of the smart
	// contract
	t.Parallel()

	ctx, txChain := NewTXTestingContext(t)
	requireT := require.New(t)

	wasmClient := wasmtypes.NewQueryClient(txChain.TXChain.ClientContext)

	owner := txChain.TXChain.GenAccount()
	trustedAddress1 := txChain.TXChain.GenAccount()
	trustedAddress2 := txChain.TXChain.GenAccount()
	trustedAddress3 := txChain.TXChain.GenAccount()

	txChain.TXChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(5000000000))),
	)

	contractClient := tx.NewContractClient(tx.DefaultContractClientConfig(nil, ""), txChain.TXChain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, tx.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold:  2,
		MinAmount:  sdkmath.NewIntFromUint64(100),
		MaxAmount:  sdkmath.NewIntFromUint64(200_000_000),
		XRPLTokens: TestXRPLTokens,
		Label:      "bank_threshold_send",
	})
	requireT.NoError(err)
	requireT.NoError(contractClient.SetContractAddress(contractAddr))

	contractInfoRes, err := wasmClient.ContractInfo(ctx, &wasmtypes.QueryContractInfoRequest{
		Address: contractAddr.String(),
	})
	requireT.NoError(err)

	t.Log("Deploying new contract.")
	// deploy new version of the contract
	newCodeID, err := contractClient.Deploy(ctx, owner)
	requireT.NoError(err)

	t.Log("Migrating the contract.")
	_, err = contractClient.MigrateContract(ctx, owner, newCodeID)
	requireT.NoError(err)

	newContractInfo, err := wasmClient.ContractInfo(ctx, &wasmtypes.QueryContractInfoRequest{
		Address: contractAddr.String(),
	})
	requireT.NoError(err)
	requireT.Equal(newCodeID, newContractInfo.ContractInfo.CodeID)
	requireT.NotEqual(contractInfoRes.ContractInfo.CodeID, newContractInfo.ContractInfo.CodeID)
}
