package factory

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/hashing/blake2b"
	"github.com/ElrondNetwork/elrond-go-core/hashing/sha256"
	"github.com/ElrondNetwork/elrond-go-crypto"
	"github.com/ElrondNetwork/elrond-go-crypto/signing"
	disabledCrypto "github.com/ElrondNetwork/elrond-go-crypto/signing/disabled"
	disabledMultiSig "github.com/ElrondNetwork/elrond-go-crypto/signing/disabled/multisig"
	disabledSig "github.com/ElrondNetwork/elrond-go-crypto/signing/disabled/singlesig"
	"github.com/ElrondNetwork/elrond-go-crypto/signing/ed25519"
	"github.com/ElrondNetwork/elrond-go-crypto/signing/ed25519/singlesig"
	"github.com/ElrondNetwork/elrond-go-crypto/signing/mcl"
	mclMultiSig "github.com/ElrondNetwork/elrond-go-crypto/signing/mcl/multisig"
	mclSig "github.com/ElrondNetwork/elrond-go-crypto/signing/mcl/singlesig"
	"github.com/ElrondNetwork/elrond-go-crypto/signing/multisig"
	"github.com/ElrondNetwork/elrond-go/config"
	"github.com/ElrondNetwork/elrond-go/consensus"
	"github.com/ElrondNetwork/elrond-go/errors"
	"github.com/ElrondNetwork/elrond-go/factory/peerSignatureHandler"
	"github.com/ElrondNetwork/elrond-go/genesis/process/disabled"
	storageFactory "github.com/ElrondNetwork/elrond-go/storage/factory"
	"github.com/ElrondNetwork/elrond-go/storage/storageUnit"
	"github.com/ElrondNetwork/elrond-go/vm"
	systemVM "github.com/ElrondNetwork/elrond-go/vm/process"
)

const disabledSigChecking = "disabled"

// CryptoComponentsFactoryArgs holds the arguments needed for creating crypto components
type CryptoComponentsFactoryArgs struct {
	ValidatorKeyPemFileName              string
	SkIndex                              int
	Config                               config.Config
	CoreComponentsHolder                 CoreComponentsHolder
	ActivateBLSPubKeyMessageVerification bool
	KeyLoader                            KeyLoaderHandler
	IsInImportMode                       bool
	ImportModeNoSigCheck                 bool
}

type cryptoComponentsFactory struct {
	consensusType                        string
	validatorKeyPemFileName              string
	skIndex                              int
	config                               config.Config
	coreComponentsHolder                 CoreComponentsHolder
	activateBLSPubKeyMessageVerification bool
	keyLoader                            KeyLoaderHandler
	isInImportMode                       bool
	importModeNoSigCheck                 bool
}

// cryptoParams holds the node public/private key data
type cryptoParams struct {
	publicKey       crypto.PublicKey
	privateKey      crypto.PrivateKey
	publicKeyString string
	publicKeyBytes  []byte
	privateKeyBytes []byte
}

// cryptoComponents struct holds the crypto components
type cryptoComponents struct {
	txSingleSigner      crypto.SingleSigner
	blockSingleSigner   crypto.SingleSigner
	multiSigner         crypto.MultiSigner
	peerSignHandler     crypto.PeerSignatureHandler
	blockSignKeyGen     crypto.KeyGenerator
	txSignKeyGen        crypto.KeyGenerator
	messageSignVerifier vm.MessageSignVerifier
	cryptoParams
}

// NewCryptoComponentsFactory returns a new crypto components factory
func NewCryptoComponentsFactory(args CryptoComponentsFactoryArgs) (*cryptoComponentsFactory, error) {
	if check.IfNil(args.CoreComponentsHolder) {
		return nil, errors.ErrNilCoreComponents
	}
	if len(args.ValidatorKeyPemFileName) == 0 {
		return nil, errors.ErrNilPath
	}
	if args.KeyLoader == nil {
		return nil, errors.ErrNilKeyLoader
	}

	ccf := &cryptoComponentsFactory{
		consensusType:                        args.Config.Consensus.Type,
		validatorKeyPemFileName:              args.ValidatorKeyPemFileName,
		skIndex:                              args.SkIndex,
		config:                               args.Config,
		coreComponentsHolder:                 args.CoreComponentsHolder,
		activateBLSPubKeyMessageVerification: args.ActivateBLSPubKeyMessageVerification,
		keyLoader:                            args.KeyLoader,
		isInImportMode:                       args.IsInImportMode,
		importModeNoSigCheck:                 args.ImportModeNoSigCheck,
	}

	return ccf, nil
}

// Create will create and return crypto components
func (ccf *cryptoComponentsFactory) Create() (*cryptoComponents, error) {
	suite, err := ccf.getSuite()
	if err != nil {
		return nil, err
	}

	blockSignKeyGen := signing.NewKeyGenerator(suite)
	cp, err := ccf.createCryptoParams(blockSignKeyGen)
	if err != nil {
		return nil, err
	}

	txSignKeyGen := signing.NewKeyGenerator(ed25519.NewEd25519())
	txSingleSigner := &singlesig.Ed25519Signer{}
	processingSingleSigner, err := ccf.createSingleSigner(false)
	if err != nil {
		return nil, err
	}

	interceptSingleSigner, err := ccf.createSingleSigner(ccf.importModeNoSigCheck)
	if err != nil {
		return nil, err
	}

	multisigHasher, err := ccf.getMultiSigHasherFromConfig()
	if err != nil {
		return nil, err
	}

	multiSigner, err := ccf.createMultiSigner(multisigHasher, cp, blockSignKeyGen, ccf.importModeNoSigCheck)
	if err != nil {
		return nil, err
	}

	var messageSignVerifier vm.MessageSignVerifier
	if ccf.activateBLSPubKeyMessageVerification {
		messageSignVerifier, err = systemVM.NewMessageSigVerifier(blockSignKeyGen, processingSingleSigner)
		if err != nil {
			return nil, err
		}
	} else {
		messageSignVerifier, err = disabled.NewMessageSignVerifier(blockSignKeyGen)
		if err != nil {
			return nil, err
		}
	}

	cacheConfig := ccf.config.PublicKeyPIDSignature
	cachePkPIDSignature, err := storageUnit.NewCache(storageFactory.GetCacherFromConfig(cacheConfig))
	if err != nil {
		return nil, err
	}

	peerSigHandler, err := peerSignatureHandler.NewPeerSignatureHandler(cachePkPIDSignature, interceptSingleSigner, blockSignKeyGen)
	if err != nil {
		return nil, err
	}

	log.Debug("block sign pubkey", "value", cp.publicKeyString)

	return &cryptoComponents{
		txSingleSigner:      txSingleSigner,
		blockSingleSigner:   interceptSingleSigner,
		multiSigner:         multiSigner,
		peerSignHandler:     peerSigHandler,
		blockSignKeyGen:     blockSignKeyGen,
		txSignKeyGen:        txSignKeyGen,
		messageSignVerifier: messageSignVerifier,
		cryptoParams:        *cp,
	}, nil
}

func (ccf *cryptoComponentsFactory) createSingleSigner(importModeNoSigCheck bool) (crypto.SingleSigner, error) {
	if importModeNoSigCheck {
		log.Warn("using disabled single signer because the node is running in import-db 'turbo mode'")
		return &disabledSig.DisabledSingleSig{}, nil
	}

	switch ccf.consensusType {
	case consensus.BlsConsensusType:
		return &mclSig.BlsSingleSigner{}, nil
	case disabledSigChecking:
		log.Warn("using disabled single signer")
		return &disabledSig.DisabledSingleSig{}, nil
	default:
		return nil, errors.ErrInvalidConsensusConfig
	}
}

func (ccf *cryptoComponentsFactory) getMultiSigHasherFromConfig() (hashing.Hasher, error) {
	if ccf.consensusType == consensus.BlsConsensusType && ccf.config.MultisigHasher.Type != "blake2b" {
		return nil, errors.ErrMultiSigHasherMissmatch
	}

	switch ccf.config.MultisigHasher.Type {
	case "sha256":
		return sha256.NewSha256(), nil
	case "blake2b":
		if ccf.consensusType == consensus.BlsConsensusType {
			return blake2b.NewBlake2bWithSize(multisig.BlsHashSize)
		}
		return blake2b.NewBlake2b(), nil
	}

	return nil, errors.ErrMissingMultiHasherConfig
}

func (ccf *cryptoComponentsFactory) createMultiSigner(
	hasher hashing.Hasher,
	cp *cryptoParams,
	blSignKeyGen crypto.KeyGenerator,
	importModeNoSigCheck bool,
) (crypto.MultiSigner, error) {
	if importModeNoSigCheck {
		log.Warn("using disabled multi signer because the node is running in import-db 'turbo mode'")
		return &disabledMultiSig.DisabledMultiSig{}, nil
	}

	switch ccf.consensusType {
	case consensus.BlsConsensusType:
		blsSigner := &mclMultiSig.BlsMultiSigner{Hasher: hasher}
		return multisig.NewBLSMultisig(blsSigner, []string{string(cp.publicKeyBytes)}, cp.privateKey, blSignKeyGen, uint16(0))
	case disabledSigChecking:
		log.Warn("using disabled multi signer")
		return &disabledMultiSig.DisabledMultiSig{}, nil
	default:
		return nil, errors.ErrInvalidConsensusConfig
	}
}

func (ccf *cryptoComponentsFactory) getSuite() (crypto.Suite, error) {
	switch ccf.config.Consensus.Type {
	case consensus.BlsConsensusType:
		return mcl.NewSuiteBLS12(), nil
	case disabledSigChecking:
		log.Warn("using disabled multi signer")
		return disabledCrypto.NewDisabledSuite(), nil
	default:
		return nil, errors.ErrInvalidConsensusConfig
	}
}

func (ccf *cryptoComponentsFactory) createCryptoParams(
	keygen crypto.KeyGenerator,
) (*cryptoParams, error) {

	if ccf.isInImportMode {
		return ccf.generateCryptoParams(keygen)
	}

	return ccf.readCryptoParams(keygen)
}

func (ccf *cryptoComponentsFactory) readCryptoParams(keygen crypto.KeyGenerator) (*cryptoParams, error) {
	cp := &cryptoParams{}
	sk, readPk, err := ccf.getSkPk()
	if err != nil {
		return nil, err
	}

	cp.privateKey, err = keygen.PrivateKeyFromByteArray(sk)
	if err != nil {
		return nil, err
	}

	cp.publicKey = cp.privateKey.GeneratePublic()
	if len(readPk) > 0 {
		cp.publicKeyBytes, err = cp.publicKey.ToByteArray()
		if err != nil {
			return nil, err
		}

		if !bytes.Equal(cp.publicKeyBytes, readPk) {
			return nil, errors.ErrPublicKeyMismatch
		}
	}

	validatorKeyConverter := ccf.coreComponentsHolder.ValidatorPubKeyConverter()
	cp.publicKeyString = validatorKeyConverter.Encode(cp.publicKeyBytes)

	return cp, nil
}

func (ccf *cryptoComponentsFactory) generateCryptoParams(keygen crypto.KeyGenerator) (*cryptoParams, error) {
	log.Warn("the node is in import mode! Will generate a fresh new BLS key")
	cp := &cryptoParams{}
	cp.privateKey, cp.publicKey = keygen.GeneratePair()

	var err error
	cp.publicKeyBytes, err = cp.publicKey.ToByteArray()
	if err != nil {
		return nil, err
	}

	validatorKeyConverter := ccf.coreComponentsHolder.ValidatorPubKeyConverter()
	cp.publicKeyString = validatorKeyConverter.Encode(cp.publicKeyBytes)

	return cp, nil
}

func (ccf *cryptoComponentsFactory) getSkPk() ([]byte, []byte, error) {
	encodedSk, pkString, err := ccf.keyLoader.LoadKey(ccf.validatorKeyPemFileName, ccf.skIndex)
	if err != nil {
		return nil, nil, err
	}

	skBytes, err := hex.DecodeString(string(encodedSk))
	if err != nil {
		return nil, nil, fmt.Errorf("%w for encoded secret key", err)
	}

	validatorKeyConverter := ccf.coreComponentsHolder.ValidatorPubKeyConverter()
	pkBytes, err := validatorKeyConverter.Decode(pkString)
	if err != nil {
		return nil, nil, fmt.Errorf("%w for encoded public key %s", err, pkString)
	}

	return skBytes, pkBytes, nil
}

// Close closes all underlying components that need closing
func (cc *cryptoComponents) Close() error {
	return nil
}
