package blockchain

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/dgraph-io/badger/v3"
)

// 2. Badger has no tables. Use prefix for utxos

var (
	utxoPrefix   = []byte("utxo-")
	prefixLength = len(utxoPrefix)
)

// 1. Create main struct for our unspent tx output set
type UTXOSet struct {
	Blockchain *BlockChain //access the database that is connected to the bc, so we can create new layer inside the db that just has the utxos
}

//3. Able to delete in bult the prefix keys of the database

func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.Blockchain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 100000 // Optimal amount of keys to delete (if there are more, will run more times)
	u.Blockchain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil) // Copy the key itself put into keys for delete piece of data
			keysForDelete = append(keysForDelete, key)
			keysCollected++ //incr counter
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					log.Panic(err)
				}
				keysForDelete = make([][]byte, 0, collectSize) //recreate an empty slice of bytes for this type
				keysCollected = 0
			}
		}
		if keysCollected > 0 { // check to see if counter is above 0 but below 100 000. If there are keys to del, del them with below statement
			if err := deleteKeys(keysForDelete); err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}

// 4. Need to modify some bc methods, as they are going to be ripped out ou the bc file and put into this file

//5. Reindex meth

func (u UTXOSet) Reindex() {
	db := u.Blockchain.Database

	u.DeleteByPrefix(utxoPrefix)

	UTXO := u.Blockchain.FindUTXO()

	err := db.Update(func(txn *badger.Txn) error { //Read into db
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}
			key = append(utxoPrefix, key...)

			err = txn.Set(key, outs.Serialize())
			Handle(err)
		}

		return nil
	})
	Handle(err)
}

// Clear out db with all prefixes attached to it and rebuild the set inside of the db

// 6. Update func

func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					updatedOuts := TxOutputs{}
					inID := append(utxoPrefix, in.ID...)
					item, err := txn.Get(inID)
					Handle(err)
					v, err := item.ValueCopy(nil) // get the value from the item with new Badger DB code
					Handle(err)

					outs := DeserializeOutputs(v)

					for outIdx, out := range outs.Outputs { //Unspent outp is not attached to an input, so we can verify that an out hasn't been spent by knowing that it hasn't been attached to an input
						if outIdx != in.Out {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						if err := txn.Delete(inID); err != nil {
							log.Panic(err)
						}

					} else {
						if err := txn.Set(inID, updatedOuts.Serialize()); err != nil {
							log.Panic(err)
						}
					}
				}
			}

			newOutputs := TxOutputs{} // For coinbase transactions. Always going to be unspent
			for _, out := range tx.Outputs {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			txID := append(utxoPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	Handle(err)
}

// 7. Helper func

func (u UTXOSet) CountTransactions() int {
	db := u.Blockchain.Database
	counter := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			counter++
		}

		return nil
	})

	Handle(err)

	return counter
}

// 8. Move FindUnspentTransactions from bc

func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput

	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			v, err := item.ValueCopy(nil)
			Handle(err)
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	Handle(err)

	return UTXOs
}

// 9. Move FindSpendableOutputs from bc

func (u UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.ValueCopy(nil) // New Badger Db code
			Handle(err)
			k = bytes.TrimPrefix(k, utxoPrefix)
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				}
			}
		}
		return nil
	})
	Handle(err)
	return accumulated, unspentOuts
}

// 10. Go to transactions.go
