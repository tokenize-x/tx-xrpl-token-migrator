//go:build integrationtests

package integrationtests

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/CoreumFoundation/coreum/v4/testutil/integration"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
)

func TestContractMigration(t *testing.T) {
	// since we don't add state specific changes we test contract migration with the same source code of the smart
	// contract
	t.Parallel()

	ctx, coreumChain := NewCoreumTestingContext(t)
	requireT := require.New(t)

	wasmClient := wasmtypes.NewQueryClient(coreumChain.ClientContext)

	owner := coreumChain.GenAccount()
	trustedAddress1 := coreumChain.GenAccount()
	trustedAddress2 := coreumChain.GenAccount()
	trustedAddress3 := coreumChain.GenAccount()

	coreumChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, coreumChain.NewCoin(sdk.NewInt(5000000000))),
	)

	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), coreumChain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold: 2,
		MinAmount: sdk.NewInt(100),
		MaxAmount: sdk.NewInt(200_000_000),
		Label:     "bank_threshold_send",
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
