package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// 4. Load file meth (load all of the wallets from our file)

func (ws *Wallets) LoadFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) { // First check if Wallet file exists or not
		return err
	}

	var wallets Wallets //Create a var for our wallets

	//Read the file

	fileContent, err := ioutil.ReadFile(walletFile)

	// Then we want to do what we did in the save file meth, backwards

	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets

	return nil
}

// Now create 'Create Wallets Func' under Wallets Struct

const walletFile = "./tmp/wallets.data" //Wallet mod is separate from the BC module. This points to where we want to store on our disk

// 1.Define the shape that our data will take (map)

type Wallets struct {
	Wallets map[string]*Wallet //address as a string which corresponds to wallet struct
	//so when user inputs their address they can fetch their wallet
}

// 5. Create Wallets Func which will populate our wallets
func CreateWallets() (*Wallets, error) {
	wallets := Wallets{}                       // Create W struct
	wallets.Wallets = make(map[string]*Wallet) // Make the map for the wallets field inside wallets struct
	err := wallets.LoadFile()                  // Load from file
	return &wallets, err                       // return wallets structure
}

// 6. Get Wallet using address
func (ws Wallets) GetWallet(address string) Wallet { // pass in the address as a string
	return *ws.Wallets[address] // return calling to our wallet struc with the address as the key
}

// 7. Can also create a func that will allow us to get all of the add in our wallet structure
func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string // array of strings
	// it through all of the wallets in our memory
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	return addresses
}

// 8. Wat to add a wallet to our wallets memory map
func (ws *Wallets) AddWallet() string {
	wallet := MakeWallet() // Call the Make Wallets Func to build a new wallet
	// Create the address (need in string format)
	address := fmt.Sprintf("%s", wallet.Address())
	ws.Wallets[address] = wallet // put wallet into wallets map with address as key
	return address
}

// 9. Now add some logic to cli, so we can print stuff (cli.go)

// 2. Create a save file meth on the wallets structure

func (ws Wallets) SaveFile() {
	var content bytes.Buffer //create bytes buffer so we can store all of the content we want to save to the file
	// Use Gob encoding lib to encode the buffers into the file
	// Need to register that we're using the e.p256 algo
	gob.Register(elliptic.P256())
	// create the new enc on the bytes buffer
	Encoder := gob.NewEncoder(&content)
	// Encode the data by calling encoder encode
	err := Encoder.Encode(ws)

	if err != nil {
		log.Panic(err)
	}

	// 3. Write all of this into the file

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644) //Pass in WF, bytes portion of content and give Read & Wrtite permission
	if err != nil {
		log.Panic(err)
	}

	// 4. Create Load File meth above

}
