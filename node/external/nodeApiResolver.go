package external

import (
	"github.com/ElrondNetwork/elrond-go/statusHandler/nodeDetails"
)

// NodeApiResolver can resolve API requests
type NodeApiResolver struct {
	scDataGetter   ScDataGetter
	detailsHandler nodeDetails.NodeDetails
}

// NewNodeApiResolver creates a new NodeApiResolver instance
func NewNodeApiResolver(scDataGetter ScDataGetter, detailsHandler nodeDetails.NodeDetails) (*NodeApiResolver, error) {
	if scDataGetter == nil || scDataGetter.IsInterfaceNil() {
		return nil, ErrNilScDataGetter
	}

	if detailsHandler == nil || detailsHandler.IsInterfaceNil() {
		return nil, ErrNilNodeDetails
	}

	return &NodeApiResolver{
		scDataGetter:   scDataGetter,
		detailsHandler: detailsHandler,
	}, nil
}

// GetVmValue retrieves data stored in a SC account through a VM
func (nar *NodeApiResolver) GetVmValue(address string, funcName string, argsBuff ...[]byte) ([]byte, error) {
	return nar.scDataGetter.Get([]byte(address), funcName, argsBuff...)
}

// NodeDetails return an implementation of the NodeDetails interface
func (nar *NodeApiResolver) NodeDetails() nodeDetails.NodeDetails {
	return nar.detailsHandler
}

// IsInterfaceNil returns true if there is no value under the interface
func (nar *NodeApiResolver) IsInterfaceNil() bool {
	if nar == nil {
		return true
	}
	return false
}
