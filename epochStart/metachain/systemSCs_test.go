package metachain

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/core"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/block"
	"github.com/ElrondNetwork/elrond-go/data/blockchain"
	"github.com/ElrondNetwork/elrond-go/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/state/factory"
	"github.com/ElrondNetwork/elrond-go/data/trie"
	"github.com/ElrondNetwork/elrond-go/dataRetriever"
	"github.com/ElrondNetwork/elrond-go/dataRetriever/dataPool"
	"github.com/ElrondNetwork/elrond-go/epochStart"
	"github.com/ElrondNetwork/elrond-go/epochStart/mock"
	"github.com/ElrondNetwork/elrond-go/genesis/process/disabled"
	"github.com/ElrondNetwork/elrond-go/hashing"
	"github.com/ElrondNetwork/elrond-go/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go/marshal"
	economics2 "github.com/ElrondNetwork/elrond-go/process/economics"
	vmFactory "github.com/ElrondNetwork/elrond-go/process/factory"
	metaProcess "github.com/ElrondNetwork/elrond-go/process/factory/metachain"
	"github.com/ElrondNetwork/elrond-go/process/peer"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/builtInFunctions"
	"github.com/ElrondNetwork/elrond-go/process/smartContract/hooks"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/ElrondNetwork/elrond-go/vm"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts"
	"github.com/ElrondNetwork/elrond-go/vm/systemSmartContracts/defaults"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemSCProcessor_ProcessSystemSmartContract(t *testing.T) {
	t.Parallel()

	args, _ := createFullArgumentsForSystemSCProcessing()
	args.ChanceComputer = &mock.ChanceComputerStub{
		GetChanceCalled: func(rating uint32) uint32 {
			if rating == 0 {
				return 10
			}
			return rating
		},
	}
	s, _ := NewSystemSCProcessor(args)

	prepareStakingContractWithData(args.UserAccountsDB, []byte("jailedPubKey0"), []byte("waitingPubKey"), args.Marshalizer)
	jailedAcc, _ := args.PeerAccountsDB.LoadAccount([]byte("jailedPubKey0"))
	_ = args.PeerAccountsDB.SaveAccount(jailedAcc)

	validatorInfos := make(map[uint32][]*state.ValidatorInfo)
	vInfo := &state.ValidatorInfo{
		PublicKey:       []byte("jailedPubKey0"),
		ShardId:         0,
		List:            string(core.JailedList),
		TempRating:      1,
		RewardAddress:   []byte("address"),
		AccumulatedFees: big.NewInt(0),
	}
	validatorInfos[0] = append(validatorInfos[0], vInfo)
	err := s.ProcessSystemSmartContract(validatorInfos)
	assert.Nil(t, err)

	assert.Equal(t, len(validatorInfos[0]), 1)
	newValidatorInfo := validatorInfos[0][0]
	assert.Equal(t, newValidatorInfo.List, string(core.NewList))
}

func TestSystemSCProcessor_JailedNodesShouldNotBeSwappedAllAtOnce(t *testing.T) {
	t.Parallel()

	args, _ := createFullArgumentsForSystemSCProcessing()
	args.ChanceComputer = &mock.ChanceComputerStub{
		GetChanceCalled: func(rating uint32) uint32 {
			if rating == 0 {
				return 10
			}
			return rating
		},
	}
	s, _ := NewSystemSCProcessor(args)
	require.NotNil(t, s)

	numEligible := 9
	numWaiting := 5
	numJailed := 8
	stakingScAcc := createStakingScAcc(args.UserAccountsDB)
	createEligibleNodes(numEligible, stakingScAcc, args.Marshalizer)
	_ = createWaitingNodes(numWaiting, stakingScAcc, args.UserAccountsDB, args.Marshalizer)
	jailed := createJailedNodes(numJailed, stakingScAcc, args.UserAccountsDB, args.PeerAccountsDB, args.Marshalizer)
	validatorsInfo := make(map[uint32][]*state.ValidatorInfo)
	validatorsInfo[0] = append(validatorsInfo[0], jailed...)

	err := s.ProcessSystemSmartContract(validatorsInfo)
	assert.Nil(t, err)
	for i := 0; i < numWaiting; i++ {
		assert.Equal(t, string(core.NewList), validatorsInfo[0][i].List)
	}
	for i := numWaiting; i < numJailed; i++ {
		assert.Equal(t, string(core.JailedList), validatorsInfo[0][i].List)
	}
}

func createStakingScAcc(accountsDB state.AccountsAdapter) state.UserAccountHandler {
	acc, _ := accountsDB.LoadAccount(vm.StakingSCAddress)
	stakingSCAcc := acc.(state.UserAccountHandler)

	return stakingSCAcc
}

func createEligibleNodes(numNodes int, stakingSCAcc state.UserAccountHandler, marshalizer marshal.Marshalizer) {
	for i := 0; i < numNodes; i++ {
		stakedData := &systemSmartContracts.StakedDataV2_0{
			Waiting:       false,
			Staked:        true,
			StakedNonce:   0,
			RewardAddress: []byte(fmt.Sprintf("rewardAddress_w%d", i)),
			StakeValue:    big.NewInt(100),
		}
		marshaledData, _ := marshalizer.Marshal(stakedData)
		stakingSCAcc.DataTrieTracker().SaveKeyValue([]byte(fmt.Sprintf("waiting_%d", i)), marshaledData)
	}
}

func createJailedNodes(numNodes int, stakingSCAcc state.UserAccountHandler, userAccounts state.AccountsAdapter, peerAccounts state.AccountsAdapter, marshalizer marshal.Marshalizer) []*state.ValidatorInfo {
	validatorInfos := make([]*state.ValidatorInfo, 0)

	for i := 0; i < numNodes; i++ {
		stakedData := &systemSmartContracts.StakedDataV2_0{
			Staked:        true,
			RewardAddress: []byte(fmt.Sprintf("rewardAddress_j%d", i)),
			StakeValue:    big.NewInt(100),
		}
		marshaledData, _ := marshalizer.Marshal(stakedData)
		stakingSCAcc.DataTrieTracker().SaveKeyValue([]byte(fmt.Sprintf("jailed__%d", i)), marshaledData)

		_ = userAccounts.SaveAccount(stakingSCAcc)

		jailedAcc, _ := peerAccounts.LoadAccount([]byte(fmt.Sprintf("jailed__%d", i)))
		_ = peerAccounts.SaveAccount(jailedAcc)

		vInfo := &state.ValidatorInfo{
			PublicKey:       []byte(fmt.Sprintf("jailed__%d", i)),
			ShardId:         0,
			List:            string(core.JailedList),
			TempRating:      1,
			RewardAddress:   []byte("address"),
			AccumulatedFees: big.NewInt(0),
		}

		validatorInfos = append(validatorInfos, vInfo)
	}

	return validatorInfos
}

func createWaitingNodes(numNodes int, stakingSCAcc state.UserAccountHandler, userAccounts state.AccountsAdapter, marshalizer marshal.Marshalizer) []*state.ValidatorInfo {
	validatorInfos := make([]*state.ValidatorInfo, 0)

	for i := 0; i < numNodes; i++ {
		stakedData := &systemSmartContracts.StakedDataV2_0{
			Waiting:       true,
			RewardAddress: []byte(fmt.Sprintf("rewardAddress_w%d", i)),
			StakeValue:    big.NewInt(100),
		}
		marshaledData, _ := marshalizer.Marshal(stakedData)
		stakingSCAcc.DataTrieTracker().SaveKeyValue([]byte(fmt.Sprintf("waiting_%d", i)), marshaledData)

		waitingKeyInList := []byte("w_" + fmt.Sprintf("waiting_%d", i))
		waitingListHead := &systemSmartContracts.WaitingList{
			FirstKey: []byte("w_" + fmt.Sprintf("waiting_%d", 0)),
			LastKey:  []byte("w_" + fmt.Sprintf("waiting_%d", numNodes-1)),
			Length:   uint32(numNodes),
		}
		marshaledData, _ = marshalizer.Marshal(waitingListHead)
		stakingSCAcc.DataTrieTracker().SaveKeyValue([]byte("waitingList"), marshaledData)

		waitingListElement := &systemSmartContracts.ElementInList{
			BLSPublicKey: []byte(fmt.Sprintf("waiting_%d", i)),
			PreviousKey:  waitingKeyInList,
			NextKey:      []byte("w_" + fmt.Sprintf("waiting_%d", i+1)),
		}
		marshaledData, _ = marshalizer.Marshal(waitingListElement)
		stakingSCAcc.DataTrieTracker().SaveKeyValue(waitingKeyInList, marshaledData)

		vInfo := &state.ValidatorInfo{
			PublicKey:       []byte(fmt.Sprintf("waiting_%d", i)),
			ShardId:         0,
			List:            string(core.WaitingList),
			TempRating:      1,
			RewardAddress:   []byte("address"),
			AccumulatedFees: big.NewInt(0),
		}

		validatorInfos = append(validatorInfos, vInfo)
	}

	_ = userAccounts.SaveAccount(stakingSCAcc)

	return validatorInfos
}

func prepareStakingContractWithData(
	accountsDB state.AccountsAdapter,
	stakedKey []byte,
	waitingKey []byte,
	marshalizer marshal.Marshalizer,
) {
	stakingSCAcc := createStakingScAcc(accountsDB)

	stakedData := &systemSmartContracts.StakedDataV2_0{
		Staked:        true,
		RewardAddress: []byte("rewardAddress"),
		StakeValue:    big.NewInt(100),
	}
	marshaledData, _ := marshalizer.Marshal(stakedData)
	stakingSCAcc.DataTrieTracker().SaveKeyValue(stakedKey, marshaledData)

	stakedData = &systemSmartContracts.StakedDataV2_0{
		Waiting:       true,
		RewardAddress: []byte("rewardAddress"),
		StakeValue:    big.NewInt(100),
	}
	marshaledData, _ = marshalizer.Marshal(stakedData)
	stakingSCAcc.DataTrieTracker().SaveKeyValue(waitingKey, marshaledData)

	waitingKeyInList := []byte("w_" + string(waitingKey))
	waitingListHead := &systemSmartContracts.WaitingList{
		FirstKey: waitingKeyInList,
		LastKey:  waitingKeyInList,
		Length:   1,
	}
	marshaledData, _ = marshalizer.Marshal(waitingListHead)
	stakingSCAcc.DataTrieTracker().SaveKeyValue([]byte("waitingList"), marshaledData)

	waitingListElement := &systemSmartContracts.ElementInList{
		BLSPublicKey: waitingKey,
		PreviousKey:  waitingKeyInList,
		NextKey:      make([]byte, 0),
	}
	marshaledData, _ = marshalizer.Marshal(waitingListElement)
	stakingSCAcc.DataTrieTracker().SaveKeyValue(waitingKeyInList, marshaledData)

	_ = accountsDB.SaveAccount(stakingSCAcc)
}

func createAccountsDB(
	hasher hashing.Hasher,
	marshalizer marshal.Marshalizer,
	accountFactory state.AccountFactory,
	trieStorageManager data.StorageManager,
) *state.AccountsDB {
	tr, _ := trie.NewTrie(trieStorageManager, marshalizer, hasher, 5)
	adb, _ := state.NewAccountsDB(tr, hasher, marshalizer, accountFactory)
	return adb
}

func createFullArgumentsForSystemSCProcessing() (ArgsNewEpochStartSystemSCProcessing, vm.SystemSCContainer) {
	hasher := sha256.Sha256{}
	marshalizer := &marshal.GogoProtoMarshalizer{}
	trieFactoryManager, _ := trie.NewTrieStorageManagerWithoutPruning(createMemUnit())
	userAccountsDB := createAccountsDB(hasher, marshalizer, factory.NewAccountCreator(), trieFactoryManager)
	peerAccountsDB := createAccountsDB(hasher, marshalizer, factory.NewPeerAccountCreator(), trieFactoryManager)

	argsValidatorsProcessor := peer.ArgValidatorStatisticsProcessor{
		Marshalizer:         marshalizer,
		NodesCoordinator:    &mock.NodesCoordinatorStub{},
		ShardCoordinator:    &mock.ShardCoordinatorStub{},
		DataPool:            &testscommon.PoolsHolderStub{},
		StorageService:      &mock.ChainStorerStub{},
		PubkeyConv:          &mock.PubkeyConverterMock{},
		PeerAdapter:         peerAccountsDB,
		Rater:               &mock.RaterStub{},
		RewardsHandler:      &mock.RewardsHandlerStub{},
		NodesSetup:          &mock.NodesSetupStub{},
		MaxComputableRounds: 1,
		EpochNotifier:       &mock.EpochNotifierStub{},
	}
	vCreator, _ := peer.NewValidatorStatisticsProcessor(argsValidatorsProcessor)

	blockChain := blockchain.NewMetaChain()
	argsHook := hooks.ArgBlockChainHook{
		Accounts:         userAccountsDB,
		PubkeyConv:       &mock.PubkeyConverterMock{},
		StorageService:   &mock.ChainStorerStub{},
		BlockChain:       blockChain,
		ShardCoordinator: &mock.ShardCoordinatorStub{},
		Marshalizer:      marshalizer,
		Uint64Converter:  &mock.Uint64ByteSliceConverterMock{},
		BuiltInFunctions: builtInFunctions.NewBuiltInFunctionContainer(),
	}

	gasSchedule := make(map[string]map[string]uint64)
	defaults.FillGasMapInternal(gasSchedule, 1)
	signVerifer, _ := disabled.NewMessageSignVerifier(&mock.KeyGenMock{})

	nodesSetup := &mock.NodesSetupStub{}
	metaVmFactory, _ := metaProcess.NewVMContainerFactory(
		argsHook,
		createEconomicsData(),
		signVerifer,
		gasSchedule,
		nodesSetup,
		hasher,
		marshalizer,
		&config.SystemSmartContractsConfig{
			ESDTSystemSCConfig: config.ESDTSystemSCConfig{
				BaseIssuingCost: "1000",
				OwnerAddress:    "aaaaaa",
			},
			GovernanceSystemSCConfig: config.GovernanceSystemSCConfig{
				ProposalCost:     "500",
				NumNodes:         100,
				MinQuorum:        50,
				MinPassThreshold: 50,
				MinVetoThreshold: 50,
			},
			StakingSystemSCConfig: config.StakingSystemSCConfig{
				GenesisNodePrice:                     "1000",
				UnJailValue:                          "10",
				MinStepValue:                         "10",
				MinStakeValue:                        "1",
				UnBondPeriod:                         1,
				StakingV2Epoch:                       1000000,
				StakeEnableEpoch:                     0,
				NumRoundsWithoutBleed:                1,
				MaximumPercentageToBleed:             1,
				BleedPercentagePerRound:              1,
				MaxNumberOfNodesForStake:             100,
				NodesToSelectInAuction:               100,
				ActivateBLSPubKeyMessageVerification: false,
				MinUnstakeTokensValue:                "1",
			},
			DelegationManagerSystemSCConfig: config.DelegationManagerSystemSCConfig{
				BaseIssuingCost:    "100",
				MinCreationDeposit: "100",
				EnabledEpoch:       0,
			},
			DelegationSystemSCConfig: config.DelegationSystemSCConfig{
				MinStakeAmount: "100",
				EnabledEpoch:   0,
				MinServiceFee:  0,
				MaxServiceFee:  100,
			},
		},
		peerAccountsDB,
		&mock.ChanceComputerStub{},
		&mock.EpochNotifierStub{},
	)

	vmContainer, _ := metaVmFactory.Create()
	systemVM, _ := vmContainer.Get(vmFactory.SystemVirtualMachine)
	args := ArgsNewEpochStartSystemSCProcessing{
		SystemVM:                systemVM,
		UserAccountsDB:          userAccountsDB,
		PeerAccountsDB:          peerAccountsDB,
		Marshalizer:             marshalizer,
		StartRating:             5,
		ValidatorInfoCreator:    vCreator,
		EndOfEpochCallerAddress: vm.EndOfEpochAddress,
		StakingSCAddress:        vm.StakingSCAddress,
		ChanceComputer:          &mock.ChanceComputerStub{},
		EpochNotifier:           &mock.EpochNotifierStub{},
		GenesisNodesConfig:      nodesSetup,
	}
	return args, metaVmFactory.SystemSmartContractContainer()
}

func createEconomicsData() *economics2.EconomicsData {
	maxGasLimitPerBlock := strconv.FormatUint(1500000000, 10)
	minGasPrice := strconv.FormatUint(10, 10)
	minGasLimit := strconv.FormatUint(10, 10)

	argsNewEconomicsData := economics2.ArgsNewEconomicsData{
		Economics: &config.EconomicsConfig{
			GlobalSettings: config.GlobalSettings{
				GenesisTotalSupply: "2000000000000000000000",
				MinimumInflation:   0,
				YearSettings: []*config.YearSetting{
					{
						Year:             0,
						MaximumInflation: 0.01,
					},
				},
			},
			RewardsSettings: config.RewardsSettings{
				LeaderPercentage:              0.1,
				DeveloperPercentage:           0.1,
				ProtocolSustainabilityAddress: "protocol",
			},
			FeeSettings: config.FeeSettings{
				MaxGasLimitPerBlock:     maxGasLimitPerBlock,
				MaxGasLimitPerMetaBlock: maxGasLimitPerBlock,
				MinGasPrice:             minGasPrice,
				MinGasLimit:             minGasLimit,
				GasPerDataByte:          "1",
				DataLimitForBaseCalc:    "10000",
			},
		},
		PenalizedTooMuchGasEnableEpoch: 0,
		EpochNotifier:                  &mock.EpochNotifierStub{},
	}
	economicsData, _ := economics2.NewEconomicsData(argsNewEconomicsData)
	return economicsData
}

func TestSystemSCProcessor_ProcessSystemSmartContractInitDelegationMgr(t *testing.T) {
	t.Parallel()

	args, _ := createFullArgumentsForSystemSCProcessing()
	s, _ := NewSystemSCProcessor(args)

	s.flagDelegationEnabled.Set()
	validatorInfos := make(map[uint32][]*state.ValidatorInfo)
	err := s.ProcessSystemSmartContract(validatorInfos)
	assert.Nil(t, err)

	acc, err := s.userAccountsDB.GetExistingAccount(vm.DelegationManagerSCAddress)
	assert.Nil(t, err)

	userAcc, _ := acc.(state.UserAccountHandler)
	assert.Equal(t, userAcc.GetOwnerAddress(), vm.DelegationManagerSCAddress)
	assert.NotNil(t, userAcc.GetCode())
	assert.NotNil(t, userAcc.GetCodeMetadata())
}

func TestSystemSCProcessor_ProcessDelegationRewardsNothingToExecute(t *testing.T) {
	t.Parallel()

	args, _ := createFullArgumentsForSystemSCProcessing()
	s, _ := NewSystemSCProcessor(args)

	localCache, _ := dataPool.NewCurrentBlockPool()
	miniBlocks := []*block.MiniBlock{
		{
			SenderShardID:   0,
			ReceiverShardID: 0,
			TxHashes:        [][]byte{[]byte("txHash")},
		},
	}

	err := s.ProcessDelegationRewards(miniBlocks, localCache)
	assert.Nil(t, err)
}

func TestSystemSCProcessor_ProcessDelegationRewardsErrors(t *testing.T) {
	t.Parallel()

	args, _ := createFullArgumentsForSystemSCProcessing()
	s, _ := NewSystemSCProcessor(args)

	localCache, _ := dataPool.NewCurrentBlockPool()
	miniBlocks := []*block.MiniBlock{
		{
			SenderShardID:   core.MetachainShardId,
			ReceiverShardID: core.MetachainShardId,
			TxHashes:        [][]byte{[]byte("txHash")},
			Type:            block.RewardsBlock,
		},
	}

	err := s.ProcessDelegationRewards(nil, localCache)
	assert.Nil(t, err)

	err = s.ProcessDelegationRewards(miniBlocks, nil)
	assert.Equal(t, err, epochStart.ErrNilLocalTxCache)

	err = s.ProcessDelegationRewards(miniBlocks, localCache)
	assert.Equal(t, err, dataRetriever.ErrTxNotFoundInBlockPool)

	rwdTx := &rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: make([]byte, len(vm.StakingSCAddress)),
		Epoch:   0,
	}
	localCache.AddTx([]byte("txHash"), rwdTx)
	copy(rwdTx.RcvAddr, vm.StakingSCAddress)
	err = s.ProcessDelegationRewards(miniBlocks, localCache)
	assert.Equal(t, err, epochStart.ErrSystemDelegationCall)

	rwdTx.RcvAddr[25] = 255
	err = s.ProcessDelegationRewards(miniBlocks, localCache)
	assert.Equal(t, err, vm.ErrUnknownSystemSmartContract)

	rwdTx.RcvAddr = vm.FirstDelegationSCAddress
	err = s.ProcessDelegationRewards(miniBlocks, localCache)
	assert.Equal(t, err, epochStart.ErrSystemDelegationCall)
}

func TestSystemSCProcessor_ProcessDelegationRewards(t *testing.T) {
	t.Parallel()

	args, scContainer := createFullArgumentsForSystemSCProcessing()
	s, _ := NewSystemSCProcessor(args)

	localCache, _ := dataPool.NewCurrentBlockPool()
	miniBlocks := []*block.MiniBlock{
		{
			SenderShardID:   core.MetachainShardId,
			ReceiverShardID: core.MetachainShardId,
			TxHashes:        [][]byte{[]byte("txHash")},
			Type:            block.RewardsBlock,
		},
	}

	rwdTx := &rewardTx.RewardTx{
		Round:   0,
		Value:   big.NewInt(100),
		RcvAddr: make([]byte, len(vm.FirstDelegationSCAddress)),
		Epoch:   0,
	}
	copy(rwdTx.RcvAddr, vm.FirstDelegationSCAddress)
	rwdTx.RcvAddr[28] = 2
	localCache.AddTx([]byte("txHash"), rwdTx)

	contract, _ := scContainer.Get(vm.FirstDelegationSCAddress)
	_ = scContainer.Add(rwdTx.RcvAddr, contract)

	err := s.ProcessDelegationRewards(miniBlocks, localCache)
	assert.Nil(t, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  vm.EndOfEpochAddress,
			Arguments:   [][]byte{big.NewInt(int64(rwdTx.Epoch)).Bytes()},
			CallValue:   big.NewInt(0),
			GasProvided: 1000000,
		},
		RecipientAddr: rwdTx.RcvAddr,
		Function:      "getRewardData",
	}

	vmOutput, err := args.SystemVM.RunSmartContractCall(vmInput)
	assert.Nil(t, err)
	assert.NotNil(t, vmOutput)

	assert.Equal(t, len(vmOutput.ReturnData), 3)
	assert.True(t, bytes.Equal(vmOutput.ReturnData[0], rwdTx.Value.Bytes()))
}
