// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ = (*withdrawalRequestMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (w WithdrawalRequest) MarshalJSON() ([]byte, error) {
	type WithdrawalRequest struct {
		Source    common.Address `json:"sourceAddress"`
		PublicKey [48]byte       `json:"validatorPublicKey"`
		Amount    hexutil.Uint64 `json:"amount"`
	}
	var enc WithdrawalRequest
	enc.Source = w.Source
	enc.PublicKey = w.PublicKey
	enc.Amount = hexutil.Uint64(w.Amount)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (w *WithdrawalRequest) UnmarshalJSON(input []byte) error {
	type WithdrawalRequest struct {
		Source    *common.Address `json:"sourceAddress"`
		PublicKey *[48]byte       `json:"validatorPublicKey"`
		Amount    *hexutil.Uint64 `json:"amount"`
	}
	var dec WithdrawalRequest
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Source != nil {
		w.Source = *dec.Source
	}
	if dec.PublicKey != nil {
		w.PublicKey = *dec.PublicKey
	}
	if dec.Amount != nil {
		w.Amount = uint64(*dec.Amount)
	}
	return nil
}
