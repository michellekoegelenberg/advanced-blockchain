package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLength = 4
	version        = byte(0x00) // byte with a val of 0 (hexadecimal repr. of 0)
)

// 1. Wallet contains Pub and Private Key
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// 8. Meth allows us to generate an address for each of our wallets
// Need version, checksum and PubKeyHash to create this address
// Concat these three values together and pass them through a base58 algo to create the address
func (w Wallet) Address() []byte {
	pubHash := PublicKeyHash(w.PublicKey)

	versionedHash := append([]byte{version}, pubHash...) // put v no (3) into slice.o.b and add it to front of PKH
	checksum := Checksum(versionedHash)                  // get cs

	fullHash := append(versionedHash, checksum...)

	address := Base58Encode(fullHash)

	fmt.Printf("pub key: %x\n", w.PublicKey)
	fmt.Printf("pub hash: %x\n", pubHash)
	fmt.Printf("address: %x\n", address)

	return address

	/*	Output 1:
		pub key: 0ca992d043809bd2c02632de2e10c3a4dfd937e0b8674632fc51c53a6e36d94fb5698060689ddf6bacd4d07d60827454223517b1aa666f186ef7bc4438f6f390
		pub hash: 3ae9b6bcc5f386d436cb09f845b153e3fea5467a
		address: 31364e574172383339777a50725252597a73464c3652694831525164457452726755
		Output 2:
		pub key: ed0585ad63561e5d0989ed9b9c23c5eadaae3f641d4b7c58258462507e662047d32181443d2902547c8397e779fae0e1f82f7068d6c68c827f03995626c47657
		pub hash: 6208f58cc08c44b9e04c4bc90cf1c03e1344d9cf
		address: 3139774d784a34424b5a57316d436873544471707544755256474537704331733369

		Address starts with the same number due to const version no. we're appending to all of our addresses
	*/

}

// 2. Func to generate new Key pair
func NewKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256() // Generate curve and define size of curve

	private, err := ecdsa.GenerateKey(curve, rand.Reader)

	if err != nil {
		log.Panic(err)
	}

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	// Concat x and y to make public key

	return *private, pub

}

// 3. Abstraction func which allows us to set them into a wallet type

func MakeWallet() *Wallet {
	private, public := NewKeyPair()
	wallet := Wallet{private, public}

	return &wallet
}

// 4. Need to start creating the address
// Get Public Key Hash by running PubKey through SHA256 and RipeMD Hashing Funcs
func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	if err != nil {
		log.Panic(err)
	}

	PublicRipMdD := hasher.Sum(nil) // don't need to concat anything

	return PublicRipMdD

}

// 5. Add version to PHK

//.6 From vPKH, creats checksum (4 bytes)
// First define cs len (const above)
//then create cs func below, which will run PKH through SHA twice and use only
//first 4 bytes (Checksum)
func Checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}

// CS will be used (concat) with original PKH and Version value (3 in e.g. output above)
// and put through a base 58 encoder algo to create address

// 7. Create base58 encode/decode funcs in utils.go then go to 8 above

// Add a comment
