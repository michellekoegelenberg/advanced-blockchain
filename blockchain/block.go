package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
)

//Modify Block to store nonce so validation func can be implemented
/* Chapter 4: Edit Block Struct (block.go): Replace 'Data' with an array of Txns
Each block must have at least one tx. Can have many.
Edit Create Block and Genisis Funcs*/
type Block struct {
	Hash         []byte
	Transactions []*Transaction
	PrevHash     []byte
	Nonce        int
}

/* Chapter 4: Our POW algo needs to consider the transactions in a block
so we need to create a new funct which allows us to use a hashing mechanism
 to provide a unique representation of all our txns combined */

func (b *Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	//Range over transactions and append
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	//Use bytes.Join to concat all these bytes together and then hash them with Sha
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
	//After this, update proof.go file (InitData)
}

//Add 0 for initial nonce (after prevHash)
//Modify CreateBlock so that it runs the pow algo on each block we create
//Execute the run func on that pow which will return the nonce and hash
//Put the nonce and hash into the block structure
//Return the block
//Now we can pass around data properly :)
func CreateBlock(txs []*Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0} //Block gets init
	pow := NewProof(block)                      //Init pow with NewProof allows us to pair a target with each new block that gets created
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func Genesis(coinbase *Transaction) *Block {
	return CreateBlock([]*Transaction{coinbase}, []byte{})
}

// The Serialize meth gives us a bytes representation of our block
// Encode block into bytes

func (b *Block) Serialize() []byte {
	var res bytes.Buffer            //Create result (of type bytes.Buffer)
	encoder := gob.NewEncoder(&res) //Create new encoder on results bytes buffer

	err := encoder.Encode(b) //Call 'Encode' on the block itself. Passes back an err. Need to do err handling

	Handle(err) //Handle err func added later

	return res.Bytes() // Return the bytes portion of our result. Gives us a bytes representation of our block
}

//Deserialize func is basically the opposite of serialize
func Deserialize(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)

	Handle(err)

	return &block
}

func Handle(err error) {
	if err != nil {
		log.Panic(err)
	}
}
