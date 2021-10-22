package slash

import (
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go/process"
	"github.com/ElrondNetwork/elrond-go/process/block/interceptedBlocks"
)

// HeaderInfo contains a HeaderHandler and its associated hash
type HeaderInfo struct {
	Header data.HeaderHandler
	Hash   []byte
}

// HeaderInfoList defines a list of HeaderInfo
type HeaderInfoList []*HeaderInfo

// SlashingResult contains the slashable data as well as the severity(slashing level)
// for a possible malicious validator
type SlashingResult struct {
	SlashingLevel ThreatLevel
	Data          HeaderInfoList
}

type slashingHeaders struct {
	slashingLevel ThreatLevel
	headers       HeaderInfoList
}

// IsIndexSetInBitmap - checks if a bit is set(1) in the given bitmap
// TODO: Move this utility function in ELROND-GO-CORE
func IsIndexSetInBitmap(index uint32, bitmap []byte) bool {
	indexOutOfBounds := index >= uint32(len(bitmap))*8
	if indexOutOfBounds {
		return false
	}

	bytePos := index / 8
	byteInMap := bitmap[bytePos]
	bitPos := index % 8
	mask := uint8(1 << bitPos)
	return (byteInMap & mask) != 0
}

func convertInterceptedDataToInterceptedHeaders(data []process.InterceptedData) ([]*interceptedBlocks.InterceptedHeader, error) {
	headers := make([]*interceptedBlocks.InterceptedHeader, 0, len(data))

	for _, d := range data {
		header, castOk := d.(*interceptedBlocks.InterceptedHeader)
		if !castOk {
			return nil, process.ErrCannotCastInterceptedDataToHeader
		}
		headers = append(headers, header)
	}

	return headers, nil
}