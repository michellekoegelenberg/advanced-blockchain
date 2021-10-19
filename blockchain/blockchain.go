package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/dgraph-io/badger/v3"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func ContinueBlockChain(address string) *BlockChain {
	if DBexists() == false {
		fmt.Println("No existing blockchain found. Create one!")
		runtime.Goexit()
	}

	var lastHash []byte

	// New Bader DB API

	opt := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opt)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Continue Blockchain func error")
	}

	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte("lh"))
		Handle(err)
		lastHash, err = item.ValueCopy(nil)
		if err != nil {
			log.Fatal(err)
			fmt.Println("Value copy error")
		}
		return err
	})

	Handle(err)
	chain := BlockChain{lastHash, db}
	return &chain
}

func InitBlockChain(address string) *BlockChain {

	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	//New Bader DB API

	opt := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opt)
	if err != nil {
		log.Fatal(err)
	}

	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created/proved")
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash

		return err

	})

	Handle(err)

	blockchain := BlockChain{lastHash, db}
	return &blockchain
}

func (chain *BlockChain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {

		item, err := txn.Get([]byte("lh"))
		Handle(err)
		//This code is updated code from Badger DB new API (and found in comments section of Part3 on YouTube Tut)

		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)
			return nil
			//New code ends here
		})
		return err
	})
	Handle(err)
	newBlock := CreateBlock(transactions, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err

	})
	Handle(err)

}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}
	return iter
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block
	//Read only transaction
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)
		//New code from updated DB

		var encodedBlock []byte
		err = item.Value(func(val []byte) error {
			encodedBlock = append([]byte{}, val...)
			return nil
		})
		//New code ends

		block = Deserialize(encodedBlock)

		return nil
	})
	Handle(err)

	iter.CurrentHash = block.PrevHash

	return block

}

// Chap 6.2 (see 6.1 below)
// remove string/address, add pubkeyhash/[]byte and replace everywhere where addr was referenced
func (chain *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTxs []Transaction
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}

				}
				if out.IsLockedWithKey(pubKeyHash) { //Update with new meth
					unspentTxs = append(unspentTxs, *tx)
				}

			}

			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.UsesKey(pubKeyHash) { // Update with new meth
						inTxID := hex.EncodeToString(in.ID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

// Repeat same process
func (chain *BlockChain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(pubKeyHash) //Here

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) { //Here
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

// Same here
func (chain *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(pubKeyHash) // Here
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount { // Here, then go to txn.go file
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts
}

// Chap 6.1 Create a few methods for the bc which will allow us to find, sign and verify txns

func (bc *BlockChain) FindTransaction(ID []byte) (Transaction, error) { //Find
	iter := bc.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction does not exist")
}

// Sign

func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs) // go through map and sign all the txns independently
}

// Verify. Do roughly the same as above, just verify

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool { // return a bool
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := bc.FindTransaction(in.ID)
		Handle(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs) // return a bool
}

// Now clean up errors created by changing txn input and output struct and removing old methods (go to 6.2 above)

//dsd
