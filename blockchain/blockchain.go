package blockchain

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/dgraph-io/badger/v3"
)

/*Chapter 4 changes in const, InitBlockChain, AddBlock, call to CreateBlock  */
const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"          //Const used to verify whether blockchain db exists
	genesisData = "First Transaction from Genesis" //Arbitrary data referenced earlier in Chap 4
)

type BlockChain struct {
	LastHash []byte //Stores last hash of last block in chain
	Database *badger.DB
}

//Part 2 of Chapter 3: Create blockchain iterator (iterate through bc in database)
//Implemented at the bottom
type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

//Chap 4: Split InitBlockChain into 2 functions (by the If statements) but first create helper func
//that tells us whether databse exists
func DBexists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

//Chap 4: Create 2nd func
func ContinueBlockChain(address string) *BlockChain {
	if DBexists() == false {
		fmt.Println("No existing blockchain found. Create one!")
		runtime.Goexit()
	}

	//Run through the same as above
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
	chain := BlockChain{lastHash, db} //Create new blockchain in memory
	return &chain                     //This way we can use it further in our app
}

func InitBlockChain(address string) *BlockChain {

	var lastHash []byte

	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit() //exit the program
	}

	//New Bader DB API

	opt := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opt)
	if err != nil {
		log.Fatal(err)
	}

	Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)         //address will mine the Genesis block and get rewarded the 100 tokens
		genesis := Genesis(cbtx)                         //Creating genesis block
		fmt.Println("Genesis created/proved")            //Proving it
		err = txn.Set(genesis.Hash, genesis.Serialize()) //Genesis hash is key for Genesis block, Serialize gen block
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
	//Execute a read-only type of txn on our BadgerDB (call the database from our blockchain by calling on chain.Database)
	//Call View (read-only), which takes in a closure with a pointer to a Badger transaction
	//Returns an err
	err := chain.Database.View(func(txn *badger.Txn) error {
		//Get the current last hash out of the database (the hash of the last block in our database)
		//Call txn.Get to get the item, then unwrap the value from item and put it into our lastHash var
		//Return error if there is one
		//Handle the first err if there is one
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
	//Create a new block with the lastHash and the data we are passing in to the AddBlock func
	newBlock := CreateBlock(transactions, lastHash)
	//With our new block now created, we want to do a read/write type txn on our db
	//so we can put the new block into the database and assign the new block's hash to our last hash key
	err = chain.Database.Update(func(txn *badger.Txn) error {
		//As we did with Genesis (use new block's hash as key and serialize newBlock itself and put that in as the value)
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)
		//Set newBlock's hash as last hash value
		err = txn.Set([]byte("lh"), newBlock.Hash)
		//Grab blockchain (chain) and lastHash field and set it equal to new block's hash
		chain.LastHash = newBlock.Hash
		return err

	})
	Handle(err)
	//Succesfully created a layer of persistence for our blockchain
	//However, lost the ability to go through our blockchain and print it out like we have before
	//All the blocks are in the data layer. Can't just print them out
}

//Part 2 of Chapter 3:
//Implementing Iterator:
//Convert Blockchain struct into blockchain iterator struct
//Create method called Iterator that has a receiver for pointer to blockchain
//returns a pointer to a BlockChainIterator
func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database} //These will become the BCI's current hash and database
	return iter
}

//Because we are starting with the last hash of our bc we will be iterating backwards through the blocks
//starting with the newest and working our way back to the genesis block
//Create next func for this

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

	iter.CurrentHash = block.PrevHash //Change iter current hash to the block's prev hash

	return block

	//Part 3 of Chapter 3:
	//Create CLI to allow user to pass in a new block and print out the blockchain
	//Step 1 is to create a CommandLine struct in main.go
}

/* Chapter 4: First step in adding these pieces of functionality is creating a way of finding
all of the unspent txns, which are assigned to an address. Unspent txns are txns that have outputs
that are not referenced by any inputs. If outputs haven't been spent, those tokens still exist for
a certain user. By counting all of the unspent outputs assigned to a certain user, we can find out
how many tokens are assigned to that user.

Create Method on BC
*/

func (chain *BlockChain) FindUnspentTransactions(address string) []Transaction {
	var unspentTxs []Transaction
	//make a map where the keys are strings and the values are slices of integers
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID) //Encode into Hexadecimal string
			//Add a label to not break out of the other for loops (just this one)
		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil { //see if output is inside of our map
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}

				}
				//We then need a way to check if output can be unlocked by the address we're searching for
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}

			}

			//Check if txn is cbtx or not. If not, iterate over txn inputs.Find other out that are ref by the inp
			//Check to see if we can unlock those outputs with out address.
			//If we can we want to put it inside our map too
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
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

func (chain *BlockChain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount { //so we can't make a txn if user doesn't have enough tokens
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts //Use these 2 values to generate the general transaction
}

//Remember to fix the AddBlock func and call to CreateBlock
//Go back into transaction.go file and create a func called new transaction
