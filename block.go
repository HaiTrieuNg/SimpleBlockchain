package main

import (
	"SpartanGold/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"time"
)

type Block struct {
	//prevBlockHash  string
	PrevBlockHash  []byte
	Target         *big.Int
	Balances       map[string]int
	NextNonce      map[string]int
	Transactions   map[string]*Transaction
	ChainLength    int
	Timestamp      time.Time
	RewardAddr     string
	CoinbaseReward int
	Proof          int
	Clients        []*Client
}

func NewBlock(rewardAddr string, prevBlock *Block, target *big.Int, coinbaseReward int) *Block {
	balances := make(map[string]int)
	nextNonce := make(map[string]int)
	chainLength := 0
	prevBlockHash := []byte{}
	if prevBlock != nil {
		prevBlockHash = prevBlock.hashVal()
		// Get the balances and nonces from the previous block, if available.
		for index, val := range prevBlock.Balances {
			balances[index] = val
		}
		for index, val := range prevBlock.NextNonce {
			nextNonce[index] = val
		}
		// Used to determine the winner between competing chains.
		// Note that this is a little simplistic -- an attacker
		// could make a long, but low-work chain.  However, this works
		// well enough for us.
		chainLength = prevBlock.ChainLength + 1
	} else {
		prevBlockHash = nil
	}

	transactions := make(map[string]*Transaction)
	timestamp := time.Now()
	newBlock := Block{PrevBlockHash: prevBlockHash, Target: target, Balances: balances, NextNonce: nextNonce, Transactions: transactions, ChainLength: chainLength, Timestamp: timestamp, RewardAddr: rewardAddr, CoinbaseReward: coinbaseReward}

	if prevBlock != nil && prevBlock.RewardAddr != "" {
		// Add the previous block's rewards to the miner who found the proof.
		winnerBalance := newBlock.balanceOf(prevBlock.RewardAddr)
		newBlock.Balances[prevBlock.RewardAddr] = winnerBalance + prevBlock.totalRewards()
	}

	return &newBlock
}

/**
 * Determines whether the block is the beginning of the chain.
 *
 * @returns {Boolean} - True if this is the first block in the chain.
 */
func (b Block) isGenesisBlock() bool {
	return b.ChainLength == 0
}

/**
 * Returns true if the hash of the block is less than the target
 * proof of work value.
 *
 * @returns {Boolean} - True if the block has a valid proof.
 */
func (b Block) hasValidProof() bool {
	h := string(b.hashVal())
	n := big.NewInt(0)

	n, ok := n.SetString(h, 16)

	if !ok {
		return false
	}

	return n.Cmp(b.Target) < 0
}

/**
 * Converts a Block into string form.  Some fields are deliberately omitted.
 * Note that Block.deserialize plus block.rerun should restore the block.
 *
 * @returns {String} - The block in JSON format.
 */
func (b Block) serialize() string {
	return string(b.toJson())
}

func (b Block) toJson() []byte { //FIX THIS
	jsonFile := []byte{}
	jsonFile = append(jsonFile, '{')
	properties := make(map[string]bool)
	properties["ChainLength"] = true
	properties["Timestamp"] = true
	if b.isGenesisBlock() {
		// The genesis block does not contain a proof or transactions,
		// but is the only block than can specify balances.
		properties["Balances"] = true
	} else {
		// Other blocks must specify transactions and proof details.
		properties["Proof"] = true
		properties["Transactions"] = true
		properties["PrevBlockHash"] = true
		properties["RewardAddr"] = true
	}

	size := reflect.ValueOf(b).NumField()
	for i := 0; i < size; i++ {
		val := reflect.ValueOf(b).Field(i)
		field := reflect.TypeOf(b).Field(i).Name
		if marshallField, err := json.Marshal((val).Interface()); err != nil {
			errors.New("Convert to Json failed.")
		} else {
			if properties[field] {
				jsonFile = append(jsonFile, '"')
				jsonFile = append(jsonFile, []byte(field)...)
				jsonFile = append(jsonFile, '"')
				jsonFile = append(jsonFile, ':')
				jsonFile = append(jsonFile, (marshallField)...)
				if i+1 != len(properties) {
					jsonFile = append(jsonFile, ',')
				}
			}
		}
	}
	jsonFile = append(jsonFile, '}')
	return jsonFile
}

/**
 * Returns the cryptographic hash of the current block.
 * The block is first converted to its serial form, so
 * any unimportant fields are ignored.
 *
 * @returns {String} - cryptographic hash of the block.
 */
func (b Block) hashVal() []byte {
	//return utils.Hash(b.serialize())
	return []byte(hex.EncodeToString(utils.Hash(b.serialize())))
}

/**
 * Returns the hash of the block as its id.
 *
 * @returns {String} - A unique ID for the block.
 */
func (b Block) getId() string {
	return string(b.hashVal())
}

/**
 * Accepts a new transaction if it is valid and adds it to the block.
 *
 * @param {Transaction} tx - The transaction to add to the block.
 * @param {Client} [client] - A client object, for logging useful messages.
 *
 * @returns {Boolean} - True if the transaction was added successfully.
 */
func (b *Block) addTransaction(tx *Transaction, client *Client) bool {
	msg := NewMsg("", "", "", "")

	if _, found := b.Transactions[tx.getId()]; found {
		msg.msg = "Duplicate transaction " + tx.getId()
		if client != nil {
			client.log(msg)
		} else { // since I couldn't pull the client for this method, I'll temporarily print it
			fmt.Println(msg.msg)
		}
		return false
	} else if tx.sig == nil {
		msg.msg = "Unsigned transaction " + tx.getId()
		if client != nil {
			client.log(msg)
		} else {
			fmt.Println(msg.msg)
		}
		return false
	} else if !tx.validSignature() {
		msg.msg = "Invalid signature for transaction " + tx.getId()
		if client != nil {
			client.log(msg)
		} else {
			fmt.Println(msg.msg)
		}
		return false
	} else if !tx.sufficientFunds(*b) {
		msg.msg = "Insufficient gold for transaction " + tx.getId()
		if client != nil {
			client.log(msg)
		} else {
			fmt.Println(msg.msg)
		}
		return false
	}

	// Checking and updating nonce value.
	// This portion prevents replay attacks.
	var nonce int
	if n, found := b.NextNonce[tx.getId()]; found {
		nonce = n
	} else {
		nonce = 0
	}

	if tx.nonce < nonce {
		msg.msg = "Replayed transaction " + tx.getId()
		if client != nil {
			client.log(msg)
		} //else {
		fmt.Printf("Replayed transaction %v \n", tx.getId())
		//}
		return false
	} else if tx.nonce > nonce {
		msg.msg = "Out of order transaction " + tx.getId()
		if client != nil {
			client.log(msg)
		} //else {
		fmt.Printf("Out of order transaction %v \n", tx.getId())
		//}
		return false
	} else {
		b.NextNonce[tx.from] = nonce + 1
	}

	// Adding the transaction to the block
	b.Transactions[tx.getId()] = tx
	//fmt.Printf("Check if added %v \n", b.Transactions[tx.getId()])

	// Taking gold from the sender
	senderBalance := b.balanceOf(tx.from)
	b.Balances[tx.from] = senderBalance - tx.totalOutput()

	// Giving gold to the specified output addresses
	for ad, gold := range tx.outputs {
		b.Balances[ad] = b.balanceOf(ad) + gold
	}

	return true
}

/**
 * When a block is received from another party, it does not include balances or a record of
 * the latest nonces for each client.  This method restores this information be wiping out
 * and re-adding all transactions.  This process also identifies if any transactions were
 * invalid due to insufficient funds or replayed transactions, in which case the block
 * should be rejected.
 *
 * @param {Block} prevBlock - The previous block in the blockchain, used for initial balances.
 *
 * @returns {Boolean} - True if the block's transactions are all valid.
 */
func (b *Block) rerun(prevBlock *Block) bool {
	// Setting balances to the previous block's balances.
	b.Balances = make(map[string]int)
	b.NextNonce = make(map[string]int)

	// Re-adding all transactions.
	txs := b.Transactions
	b.Transactions = make(map[string]*Transaction)

	for ad, value := range prevBlock.Balances {
		b.Balances[ad] = value
	}
	for ad, value := range prevBlock.NextNonce {
		b.NextNonce[ad] = value
	}

	// Adding coinbase reward for prevBlock.
	winnerBalance := b.balanceOf(prevBlock.RewardAddr)
	if prevBlock.RewardAddr != "" {
		b.Balances[prevBlock.RewardAddr] = winnerBalance + prevBlock.totalRewards()
	}

	for _, value := range txs {
		success := b.addTransaction(value, nil) //not sure how to pull the client for addTransaction method
		if !success {
			fmt.Printf("Transaction %v failed.\n", value.getId())
			return false
		}
	}

	return true
}

/**
 * The total amount of gold paid to the miner who produced this block,
 * if the block is accepted.  This includes both the coinbase transaction
 * and any transaction fees.
 *
 * @returns {Number} Total reward in gold for the user.
 *
 */
func (b Block) totalRewards() int {
	total := b.CoinbaseReward
	for _, tx := range b.Transactions {
		total += tx.fee
	}
	return total
}

/**
 * Gets the available gold of a user identified by an address.
 * Note that this amount is a snapshot in time - IF the block is
 * accepted by the network, ignoring any pending transactions,
 * this is the amount of funds available to the client.
 *
 * @param {String} addr - Address of a client.
 *
 * @returns {Number} - The available gold for the specified user.
 */
func (b Block) balanceOf(addrress string) int {

	if val, ok := b.Balances[addrress]; ok {
		//fmt.Printf("%v has balance %v", val, ok)
		return val
	}
	return 0
}

/**
 * Determines whether a transaction is in the block.  Note that only the
 * block itself is checked; if it returns false, the transaction might
 * still be included in one of its ancestor blocks.
 *
 * @param {Transaction} tx - The transaction that we are checking for.
 *
 * @returns {boolean} - True if the transaction is contained in this block.
 */
func (b Block) contains(tx *Transaction) bool {
	/*if found, ok := b.transactions[tx.getId()]; ok{
		return true;
	}
	return false;*/

	_, found := b.Transactions[tx.getId()]
	if found {
		return true
	} else {
		return false
	}
}
