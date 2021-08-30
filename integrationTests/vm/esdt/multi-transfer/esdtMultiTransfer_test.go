package multitransfer

import (
	"math/big"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go/integrationTests"
	"github.com/ElrondNetwork/elrond-go/integrationTests/vm/esdt"
	"github.com/ElrondNetwork/elrond-go/testscommon/txDataBuilder"
)

const NR_ROUNDS_CROSS_SHARD = 15
const NR_ROUNDS_SAME_SHARD = 1

type esdtTransfer struct {
	tokenIdentifier string
	nonce           int64
	amount          int64
}

func TestESDTMultiTransferToVault(t *testing.T) {
	if testing.Short() {
		t.Skip("this is not a short test")
	}

	nodes, idxProposers := esdt.CreateNodesAndPrepareBalances(1)

	expectedIssuerBalance := make(map[string]map[int64]int64)
	expectedVaultBalance := make(map[string]map[int64]int64)

	defer func() {
		for _, n := range nodes {
			_ = n.Messenger.Close()
		}
	}()

	initialVal := big.NewInt(10000000000)
	integrationTests.MintAllNodes(nodes, initialVal)

	round := uint64(0)
	nonce := uint64(0)
	round = integrationTests.IncrementAndPrintRound(round)
	nonce++

	// deploy vault SC
	vaultScAddress := esdt.DeployNonPayableSmartContract(t, nodes, idxProposers, &nonce, &round,
		"../testdata/vault.wasm")

	// issue two fungible tokens
	fungibleTokenIdentifier1 := issueFungibleToken(t, nodes, idxProposers, &nonce, &round, 1000, "FUNG1")
	fungibleTokenIdentifier2 := issueFungibleToken(t, nodes, idxProposers, &nonce, &round, 1000, "FUNG2")

	expectedIssuerBalance[fungibleTokenIdentifier1] = make(map[int64]int64)
	expectedIssuerBalance[fungibleTokenIdentifier2] = make(map[int64]int64)
	expectedVaultBalance[fungibleTokenIdentifier1] = make(map[int64]int64)
	expectedVaultBalance[fungibleTokenIdentifier2] = make(map[int64]int64)

	expectedIssuerBalance[fungibleTokenIdentifier1][0] = 1000
	expectedIssuerBalance[fungibleTokenIdentifier2][0] = 1000

	// issue two NFT, with multiple NFTCreate
	nonFungibleTokenIdentifier1 := issueNft(t, nodes, idxProposers, &nonce, &round, "NFT1", false)
	nonFungibleTokenIdentifier2 := issueNft(t, nodes, idxProposers, &nonce, &round, "NFT2", false)

	expectedIssuerBalance[nonFungibleTokenIdentifier1] = make(map[int64]int64)
	expectedIssuerBalance[nonFungibleTokenIdentifier2] = make(map[int64]int64)

	expectedVaultBalance[nonFungibleTokenIdentifier1] = make(map[int64]int64)
	expectedVaultBalance[nonFungibleTokenIdentifier2] = make(map[int64]int64)

	for i := int64(1); i <= 10; i++ {
		createNFT(t, nodes, idxProposers, nonFungibleTokenIdentifier1, i, &nonce, &round)
		createNFT(t, nodes, idxProposers, nonFungibleTokenIdentifier2, i, &nonce, &round)

		expectedIssuerBalance[nonFungibleTokenIdentifier1][i] = 1
		expectedIssuerBalance[nonFungibleTokenIdentifier2][i] = 1
	}

	/*

		// issue two SFTs, with two NFTCreate for each
		semiFungibleTokenIdentifier1 := issueNft(t, nodes, idxProposers, &nonce, &round, "SFT1", true)
		semiFungibleTokenIdentifier2 := issueNft(t, nodes, idxProposers, &nonce, &round, "SFT2", true)

		expectedIssuerBalance[semiFungibleTokenIdentifier1] = make(map[int64]int64)
		expectedIssuerBalance[semiFungibleTokenIdentifier2] = make(map[int64]int64)

		expectedVaultBalance[semiFungibleTokenIdentifier1] = make(map[int64]int64)
		expectedVaultBalance[semiFungibleTokenIdentifier2] = make(map[int64]int64)

		for i := int64(1); i <= 2; i++ {
			createSFT(t, nodes, idxProposers, semiFungibleTokenIdentifier1, i, 1000, &nonce, &round)
			createSFT(t, nodes, idxProposers, semiFungibleTokenIdentifier2, i, 1000, &nonce, &round)

			expectedIssuerBalance[semiFungibleTokenIdentifier1][i] = 1000
			expectedIssuerBalance[semiFungibleTokenIdentifier2][i] = 1000
		}

	*/

	// send a single ESDT with multi-transfer
	transfers := []esdtTransfer{
		{
			tokenIdentifier: fungibleTokenIdentifier1,
			nonce:           0,
			amount:          100,
		}}
	multiTransferToVault(t, nodes, idxProposers,
		vaultScAddress, transfers,
		expectedIssuerBalance, expectedVaultBalance,
		&nonce, &round,
	)

	// send two identical transfers with multi-transfer
	transfers = []esdtTransfer{
		{
			tokenIdentifier: fungibleTokenIdentifier1,
			nonce:           0,
			amount:          50,
		},
		{
			tokenIdentifier: fungibleTokenIdentifier1,
			nonce:           0,
			amount:          50,
		}}
	multiTransferToVault(t, nodes, idxProposers,
		vaultScAddress, transfers,
		expectedIssuerBalance, expectedVaultBalance,
		&nonce, &round,
	)

	// send two different transfers amounts, same token
	transfers = []esdtTransfer{
		{
			tokenIdentifier: fungibleTokenIdentifier1,
			nonce:           0,
			amount:          50,
		},
		{
			tokenIdentifier: fungibleTokenIdentifier1,
			nonce:           0,
			amount:          100,
		}}
	multiTransferToVault(t, nodes, idxProposers,
		vaultScAddress, transfers,
		expectedIssuerBalance, expectedVaultBalance,
		&nonce, &round,
	)

	// send two different tokens, same amount
	transfers = []esdtTransfer{
		{
			tokenIdentifier: fungibleTokenIdentifier1,
			nonce:           0,
			amount:          100,
		},
		{
			tokenIdentifier: fungibleTokenIdentifier2,
			nonce:           0,
			amount:          100,
		}}
	multiTransferToVault(t, nodes, idxProposers,
		vaultScAddress, transfers,
		expectedIssuerBalance, expectedVaultBalance,
		&nonce, &round,
	)

	// send single NFT
	transfers = []esdtTransfer{
		{
			tokenIdentifier: nonFungibleTokenIdentifier1,
			nonce:           1,
			amount:          1,
		}}
	multiTransferToVault(t, nodes, idxProposers,
		vaultScAddress, transfers,
		expectedIssuerBalance, expectedVaultBalance,
		&nonce, &round,
	)

	/*

		// send two NFTs, same token ID
		transfers = []esdtTransfer{
			{
				tokenIdentifier: nonFungibleTokenIdentifier1,
				nonce:           2,
				amount:          1,
			},
			{
				tokenIdentifier: nonFungibleTokenIdentifier1,
				nonce:           3,
				amount:          1,
			}}
		multiTransferToVault(t, nodes, idxProposers,
			vaultScAddress, transfers,
			expectedIssuerBalance, expectedVaultBalance,
			&nonce, &round,
		)

		// send two NFTs, different token ID
		transfers = []esdtTransfer{
			{
				tokenIdentifier: nonFungibleTokenIdentifier1,
				nonce:           4,
				amount:          1,
			},
			{
				tokenIdentifier: nonFungibleTokenIdentifier2,
				nonce:           1,
				amount:          1,
			}}
		multiTransferToVault(t, nodes, idxProposers,
			vaultScAddress, transfers,
			expectedIssuerBalance, expectedVaultBalance,
			&nonce, &round,
		)

		// send fours NFTs, two of each different token ID
		transfers = []esdtTransfer{
			{
				tokenIdentifier: nonFungibleTokenIdentifier1,
				nonce:           5,
				amount:          1,
			},
			{
				tokenIdentifier: nonFungibleTokenIdentifier2,
				nonce:           2,
				amount:          1,
			},
			{
				tokenIdentifier: nonFungibleTokenIdentifier1,
				nonce:           6,
				amount:          1,
			},
			{
				tokenIdentifier: nonFungibleTokenIdentifier2,
				nonce:           3,
				amount:          1,
			}}
		multiTransferToVault(t, nodes, idxProposers,
			vaultScAddress, transfers,
			expectedIssuerBalance, expectedVaultBalance,
			&nonce, &round,
		)

	*/
}

func TestESDTMultiTransferAsync(t *testing.T) {
	net := integrationTests.NewTestNetworkSized(t, 2, 1, 1)
	net.Start()
	initialVal := uint64(1000000000)
	net.MintNodeAccountsUint64(initialVal)
	net.Step()

	owner := net.NodesSharded[0][0].OwnAccount
	forwarder := net.DeploySC(owner, "../testdata/forwarder.wasm")

	// Create the fungible token
	supply := int64(1000)
	fungibleTokenIdentifier := issueFungibleToken(
		t,
		net.Nodes,
		net.Proposers,
		&net.Nonce,
		&net.Round,
		supply,
		"FUNG1")

	// Send half of the tokens to the forwarder SC
	acceptMultiTransferEndpointName := "accept_funds_multi_transfer"
	txData := buildEsdtMultiTransferTxData(forwarder,
		[]esdtTransfer{
			{
				tokenIdentifier: fungibleTokenIdentifier,
				nonce:           0,
				amount:          supply / 2,
			},
		},
		acceptMultiTransferEndpointName,
	)
	tx := net.CreateTxUint64(
		owner,
		owner.Address,
		0,
		[]byte(txData))
	tx.GasLimit = net.MaxGasLimit
	_ = net.SignAndSendTx(owner, tx)
	net.Steps(NR_ROUNDS_CROSS_SHARD)
	esdt.CheckAddressHasESDTTokens(t, owner.Address, net.Nodes, fungibleTokenIdentifier, supply/2)
	esdt.CheckAddressHasESDTTokens(t, forwarder, net.Nodes, fungibleTokenIdentifier, supply/2)

	defer net.Close()
}

func issueFungibleToken(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposers []int,
	nonce *uint64, round *uint64, initialSupply int64, ticker string) string {

	tokenIssuer := nodes[0]

	esdt.IssueTestToken(nodes, initialSupply, ticker)
	waitForOperationCompletion(t, nodes, idxProposers, NR_ROUNDS_CROSS_SHARD, nonce, round)

	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))

	esdt.CheckAddressHasTokens(t, tokenIssuer.OwnAccount.Address, nodes,
		tokenIdentifier, 0, initialSupply)

	return tokenIdentifier
}

func issueNft(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposers []int,
	nonce *uint64, round *uint64, ticker string, semiFungible bool) string {

	tokenType := core.NonFungibleESDT
	if semiFungible {
		tokenType = core.SemiFungibleESDT
	}

	esdt.IssueNFT(nodes, tokenType, ticker)
	waitForOperationCompletion(t, nodes, idxProposers, NR_ROUNDS_CROSS_SHARD, nonce, round)

	issuerAddress := nodes[0].OwnAccount.Address
	tokenIdentifier := string(integrationTests.GetTokenIdentifier(nodes, []byte(ticker)))

	esdt.SetRoles(nodes, issuerAddress, []byte(tokenIdentifier), [][]byte{
		[]byte("ESDTRoleNFTCreate"),
	})
	waitForOperationCompletion(t, nodes, idxProposers, NR_ROUNDS_CROSS_SHARD, nonce, round)

	return tokenIdentifier
}

func createSFT(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposers []int,
	tokenIdentifier string, createdTokenNonce int64, initialSupply int64,
	nonce *uint64, round *uint64) {

	issuerAddress := nodes[0].OwnAccount.Address

	tokenName := "token"
	royalties := big.NewInt(0)
	hash := "someHash"
	attributes := "cool nft"
	uri := "www.my-cool-nfts.com"

	txData := txDataBuilder.NewBuilder()
	txData.Func(core.BuiltInFunctionESDTNFTCreate)
	txData.Str(tokenIdentifier)
	txData.Int64(initialSupply)
	txData.Str(tokenName)
	txData.BigInt(royalties)
	txData.Str(hash)
	txData.Str(attributes)
	txData.Str(uri)

	integrationTests.CreateAndSendTransaction(nodes[0],
		nodes,
		big.NewInt(0),
		issuerAddress,
		txData.ToString(),
		integrationTests.AdditionalGasLimit)
	waitForOperationCompletion(t, nodes, idxProposers, NR_ROUNDS_SAME_SHARD, nonce, round)

	esdt.CheckAddressHasTokens(t, issuerAddress, nodes,
		tokenIdentifier, createdTokenNonce, initialSupply)
}

func createNFT(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposers []int,
	tokenIdentifier string, createdTokenNonce int64,
	nonce *uint64, round *uint64) {

	createSFT(t, nodes, idxProposers, tokenIdentifier, createdTokenNonce, 1, nonce, round)
}

func buildEsdtMultiTransferTxData(receiverAddress []byte, transfers []esdtTransfer,
	endpointName string, arguments ...[]byte) string {

	nrTransfers := len(transfers)

	txData := txDataBuilder.NewBuilder()
	txData.Func(core.BuiltInFunctionMultiESDTNFTTransfer)
	txData.Bytes(receiverAddress)
	txData.Int(nrTransfers)

	for _, transfer := range transfers {
		txData.Str(transfer.tokenIdentifier)
		txData.Int64(transfer.nonce)
		txData.Int64(transfer.amount)
	}

	if len(endpointName) > 0 {
		txData.Str(endpointName)

		for _, arg := range arguments {
			txData.Bytes(arg)
		}
	}

	return txData.ToString()
}

func waitForOperationCompletion(t *testing.T, nodes []*integrationTests.TestProcessorNode, idxProposers []int,
	roundsToWait int, nonce *uint64, round *uint64) {

	time.Sleep(time.Second)
	*nonce, *round = integrationTests.WaitOperationToBeDone(t, nodes, roundsToWait, *nonce, *round, idxProposers)
	time.Sleep(time.Second)
}

func multiTransferToVault(
	t *testing.T,
	nodes []*integrationTests.TestProcessorNode,
	idxProposers []int,
	vaultScAddress []byte,
	transfers []esdtTransfer,
	userBalances map[string]map[int64]int64,
	scBalances map[string]map[int64]int64,
	nonce *uint64,
	round *uint64,
) {
	acceptMultiTransferEndpointName := "accept_funds_multi_transfer"
	tokenIssuerAddress := nodes[0].OwnAccount.Address

	txData := buildEsdtMultiTransferTxData(vaultScAddress,
		transfers,
		acceptMultiTransferEndpointName,
	)

	integrationTests.CreateAndSendTransaction(
		nodes[0],
		nodes,
		big.NewInt(0),
		tokenIssuerAddress,
		txData,
		integrationTests.AdditionalGasLimit,
	)
	waitForOperationCompletion(t, nodes, idxProposers, NR_ROUNDS_SAME_SHARD, nonce, round)

	// update expected balances after transfers
	for _, transfer := range transfers {
		userBalances[transfer.tokenIdentifier][transfer.nonce] -= transfer.amount
		scBalances[transfer.tokenIdentifier][transfer.nonce] += transfer.amount
	}

	// check expected vs actual values
	for _, transfer := range transfers {
		expectedUserBalance := userBalances[transfer.tokenIdentifier][transfer.nonce]
		expectedScBalance := scBalances[transfer.tokenIdentifier][transfer.nonce]

		esdt.CheckAddressHasTokens(t, tokenIssuerAddress, nodes,
			transfer.tokenIdentifier, transfer.nonce, expectedUserBalance)
		esdt.CheckAddressHasTokens(t, vaultScAddress, nodes,
			transfer.tokenIdentifier, transfer.nonce, expectedScBalance)
	}
}
