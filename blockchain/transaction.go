package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/michellekoegelenberg/advanced-blockchain/wallet"
)

/*
Chap 6.8 Replace all the places whhere we've created txn inputs and outputs (after we've removed errors in bc.go)
Start with CoinbaseTx */

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

//1. Like the block serialise one in block file
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// 2. Use as Tx ID
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{} // Empty out Tx's ID

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// 3. Need methods to to sign and verify transactions
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() { // See if coinbase (no need to sign coinbase)
		return
	}

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy() // Copy of out Tx

	for inId, in := range txCopy.Inputs {
		prevTX := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil                            // Double check
		txCopy.Inputs[inId].PubKey = prevTX.Outputs[in.Out].PubKeyHash // Each input signed separately
		txCopy.ID = txCopy.Hash()                                      // The data we're actually going to sign
		txCopy.Inputs[inId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		Handle(err)
		signature := append(r.Bytes(), s.Bytes()...) // Two numbers need to be converted into bytes

		tx.Inputs[inId].Signature = signature // Into sig field of our input

	}
}

// 4. Create verify and trimmed copy method

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool { // Verify each of the transactions
	if tx.IsCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Previous transaction not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256() // Reconstruct curve

	for inId, in := range tx.Inputs { // Identical to the piece in Sign method. During verif. need the same data that was signed.
		prevTx := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTx.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		// Unpack all the data. Deconstruct

		r := big.Int{}
		s := big.Int{}

		sigLen := len(in.Signature)
		r.SetBytes(in.Signature[:(sigLen / 2)]) //Devide sig in half (last half r, first half s)
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{} // Do the same for x and y
		y := big.Int{}
		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y} // Create new public key
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}
	// If all pass test, return true
	return true
}

// 5. add TrimmedCopy

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TxOutput{out.Value, out.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

// 5. Convert tx into string rep to see in command line
func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:     %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// 6. Move over to blockchain.go file

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txin := TxInput{[]byte{}, -1, nil, []byte(data)} // Chap 6.8 (Four fields instead of three)
	txout := NewTXOutput(100, to)                    // 6.8 replace with new func

	tx := Transaction{nil, []TxInput{txin}, []TxOutput{*txout}} // 6.8 turn into pointer, go to NewTx func below
	tx.SetID()

	return &tx
}

func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	//6.9 Connect our wallets module
	wallets, err := wallet.CreateWallets()
	Handle(err)
	w := wallets.GetWallet(from)
	pubKeyHash := wallet.PublicKeyHash(w.PublicKey)

	acc, validOutputs := chain.FindSpendableOutputs(pubKeyHash, amount) //6.9 Replace from with pubkeyhash

	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		Handle(err)

		for _, out := range outs {
			input := TxInput{txID, out, nil, w.PublicKey} //Replace old one with one with 4 fields. Final field pk of the wallet we instantiated
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to)) // NewTxO

	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from)) //NewTxo
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash() //Change SetID to hash and signTxn
	chain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]

}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}
