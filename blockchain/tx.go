package blockchain

import (
	"bytes"

	"github.com/michellekoegelenberg/advanced-blockchain/wallet"
)

type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

type TxInput struct {
	ID        []byte
	Out       int
	Signature []byte
	PubKey    []byte
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.PublicKeyHash(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (out *TxOutput) Lock(address []byte) { //Lock output
	pubKeyHash := wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool { // See if Output has been locked with key
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

/* So an output is locked if it contains a public key hash and we can unlock the output if the
PKH of a user we're passing in is the same as the PKH of the transaction's output
*/

// Lock the outputs we create and convert string into []byte
func NewTXOutput(value int, address string) *TxOutput {
	txo := &TxOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}
