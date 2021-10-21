package blockchain

import (
	"github.com/dgraph-io/badger/v3"
)

// Moved BC It struct and funcs out of BC.go file to clean up a bit
type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
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
		//New code from updated Badger DB API starts

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
