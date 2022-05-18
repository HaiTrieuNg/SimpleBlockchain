package main

import (
	"errors"
	"fmt"
	"math/big"
)

// Network message constants
const MISSING_BLOCK = "MISSING_BLOCK"

const POST_TRANSACTION = "POST_TRANSACTION"

const PROOF_FOUND = "PROOF_FOUND"

const START_MINING = "START_MINING"

// Constants for mining
const NUM_ROUNDS_MINING = 2000

// Constants related to proof-of-work target
//const POW_BASE_TARGET = new BigInteger("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16);
//const POW_BASE_TARGET, valid = new(big.Int).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16);
const POW_BASE_TARGET = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" //64
const POW_LEADING_ZEROES = 15

// Constants for mining rewards and default transaction fees
const COINBASE_AMT_ALLOWED = 25

const DEFAULT_TX_FEE = 1

// If a block is 6 blocks older than the current block, it is considered
// confirmed, for no better reason than that is what Bitcoin does.
// Note that the genesis block is always considered to be confirmed.
const CONFIRMED_DEPTH = 6

/**
 * The Blockchain class tracks configuration information and settings for the blockchain,
 * as well as some utility methods to allow for easy extensibility.
 */
func GET_MISSING_BLOCK() string    { return MISSING_BLOCK }
func GET_POST_TRANSACTION() string { return POST_TRANSACTION }
func GET_PROOF_FOUND() string      { return PROOF_FOUND }
func GET_START_MINING() string     { return START_MINING }

func GET_NUM_ROUNDS_MINING() int { return NUM_ROUNDS_MINING }

// Configurable properties.
/*func GET_POW_TARGET(b BlockChain) *big.Int {
	powT := new(big.Int)
	powT, valid := powT.SetString(POW_BASE_TARGET, 16)
	if valid {
		powT.Rsh(powT, uint(POW_LEADING_ZEROES)) //rightshift
	}
	b.cfg.powTarget = powT
	return b.cfg.powTarget
}*/
func GET_POW_TARGET(b BlockChain) *big.Int      { return b.cfg.powTarget }
func GET_COINBASE_AMT_ALLOWED(b BlockChain) int { return b.cfg.coinbaseAmount }
func GET_DEFAULT_TX_FEE(b BlockChain) int       { return b.cfg.defaultTxFee }
func GET_CONFIRMED_DEPTH(b BlockChain) int      { return b.cfg.confirmedDepth }

type BlockChain struct {
	cfg Cfg
}

type Cfg struct {
	coinbaseAmount int
	defaultTxFee   int
	confirmedDepth int
	powTarget      *big.Int
}

/*
func NewBlockChain() BlockChain {
	var cfg Cfg
	var bc BlockChain
	bc.cfg = cfg
	return bc
}
*/
/**
 * Produces a new genesis block, giving the specified clients
 * the specified amount of starting gold.  Either clientBalanceMap
 * OR startingBalances can be specified, but not both.
 *
 * If clientBalanceMap is specified, then this method will also
 * set the genesis block for every client passed in.  This option
 * is useful in single-threaded mode.
 *
 * @param {Object} cfg - Settings for the blockchain.
 * @param {Class} cfg.blockClass - Implementation of the Block class.
 * @param {Class} cfg.transactionClass - Implementation of the Transaction class.
 * @param {Map} [cfg.clientBalanceMap] - Mapping of clients to their starting balances.
 * @param {Object} [cfg.startingBalances] - Mapping of client addresses to their starting balances.
 * @param {number} [cfg.powLeadingZeroes] - Number of leading zeroes required for a valid proof-of-work.
 * @param {number} [cfg.coinbaseAmount] - Amount of gold awarded to a miner for creating a block.
 * @param {number} [cfg.defaultTxFee] - Amount of gold awarded to a miner for accepting a transaction,
 *    if not overridden by the client.
 * @param {number} [cfg.confirmedDepth] - Number of blocks required after a block before it is
 *    considered confirmed.
 *
 * @returns {Block} - The genesis block.
 */
func (b *BlockChain) makeGenesis(clientBalanceMap map[*Client]int, startingBalances map[string]int) *Block {

	if clientBalanceMap != nil && startingBalances != nil {
		errors.New("You may set clientBalanceMap OR set startingBalances, but not both.")
	}

	// Setting blockchain configuration
	b.cfg.coinbaseAmount = COINBASE_AMT_ALLOWED
	fmt.Println(b.cfg.coinbaseAmount)
	b.cfg.defaultTxFee = DEFAULT_TX_FEE
	b.cfg.confirmedDepth = CONFIRMED_DEPTH
	
	powT := new(big.Int)
	powT, valid := powT.SetString(POW_BASE_TARGET, 16)
	if valid {
		powT.Rsh(powT, uint(POW_LEADING_ZEROES)) //rightshift
	}
	b.cfg.powTarget = powT
	fmt.Println(b.cfg.powTarget)
	fmt.Println(&b.cfg.powTarget)

	// If startingBalances was specified, we initialize our balances to that object.
	balances := make(map[string]int) //empty
	if startingBalances != nil {
		balances = startingBalances
	}

	// If clientBalanceMap was initialized instead, we copy over those values
	if clientBalanceMap != nil {
		for client, balance := range clientBalanceMap {
			balances[client.address] = balance
		}
	}

	g := b.makeBlock("", nil, b.cfg.powTarget, &b.cfg.coinbaseAmount)
	// Initializing starting balances in the genesis block.
	for address, balance := range balances {
		g.Balances[address] = balance
	}

	// If clientBalanceMap was specified, we set the genesis block for every client.
	if clientBalanceMap != nil {
		for client, _ := range clientBalanceMap {
			client.setGenesisBlock(g)
			client.blockChain = b
		}
	}
	return g
}

/**
 * Converts a string representation of a block to a new Block instance.
 *
 * @param {Object} o - An object representing a block, but not necessarily an instance of Block.
 *
 * @returns {Block}
 */
func (b BlockChain) deserializeBlock(o []byte) *Block {
	//need to implement
	return nil
}

func (bc BlockChain) makeBlock(s string, b *Block, i *big.Int, c *int) *Block {
	target := i
	reward := c
	if i == nil {
		target = bc.cfg.powTarget
	}
	if c == nil {
		reward = &bc.cfg.coinbaseAmount
	}
	return NewBlock(s, b, target, *reward)
}
func makeTransaction(tx *Transaction) *Transaction {
	//NewTransaction(from string, nonce int, pubKey *rsa.PublicKey, sig []byte, fee int, outputs []Output, data string)
	//in case it's not Transaction type?
	//ntx := N (tx.from, tx.nonce, tx.pubKey, tx.sig, tx.fee, tx.outputs, tx.data)
	//return ntx
	if tx == nil {
		return nil
	} else {
		ntx := NewTransaction(tx.from, tx.nonce, tx.pubKey, tx.sig, tx.fee, tx.outputs, "")
		return ntx
	}
}
