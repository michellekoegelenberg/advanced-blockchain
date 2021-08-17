package blockchain

type TxOutput struct {
	Value  int    // value in tokens which is assigned and locked inside this output
	PubKey string // Unlocks the (tokens inside the) Value field. Usually derived via script (lang). Keeping it simple for now. Arb key to repres user address
}

/*Outputs: Indivisible. Can't reference a part of an output.
If there are 10 tokens inside our output we need to create two new outputs,
one with 5 tokens inside and another with another 5.
*/

//Inputs are just references to prev outputs

type TxInput struct {
	ID  []byte // ID references the transaction the output is inside of
	Out int    // Index of the output (within the transaction)
	Sig string // Provides data used in output's pubkey ("Jack" unlock the output being referenced by the input)
}

/* In Genesis block we have our first transaction (Coinbase Transaction)
In this transaction: One input and one output.
Input references an empty output (no older outputs).
Doesn't store sig. Stores arb data.
Reward attached to it. Released to a single account when that individual mines the coinbase.
Just adding a const to make things simple for now
*/

//Create 2 methods we need to unlock the data inside of our outputs and inputs

func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.PubKey == data
}

/* If these come back as true it means the account (data) owns the information inside of the output
or it owns the info inside of the output that is referenced by the input */
