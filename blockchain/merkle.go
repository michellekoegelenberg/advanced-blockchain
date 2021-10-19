package blockchain

import "crypto/sha256"

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct { //recursive (reference other structs)
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := MerkleNode{}

	//see if l and r nodes exist
	//concat, sha, put into data field
	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right

	return &node
}

//look at schema for merkle tree

func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	if len(data)%2 != 0 { //see if data is even, if not make it even
		data = append(data, data[len(data)-1])
	}
	//start with bottom leaves
	for _, dat := range data {
		node := NewMerkleNode(nil, nil, dat)
		nodes = append(nodes, *node)
	}

	//iter and connect into tree shape

	for i := 0; i < len(data)/2; i++ {
		var level []MerkleNode //create levels

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			level = append(level, *node) //create levels
		}

		nodes = level
	}

	tree := MerkleTree{&nodes[0]} //create the tree

	return &tree
}

// Now add to block struct, so tha blocks will be represented using the merkle tree stuct
//Go to block.go
