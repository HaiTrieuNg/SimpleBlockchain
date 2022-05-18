package main

import (
	"errors"
	"fmt"
	"time"
)

type Miner struct {
	MiningRounds int
	MClient      *Client
	CurrentBlock *Block
	Transactions []*Transaction
}

func NewMiner(name string, net *fake_net, startingBlock *Block) *Miner {
	var m Miner
	asClient := NewClient(name, net, startingBlock) //super
	m.MClient = asClient
	m.MiningRounds = NUM_ROUNDS_MINING
	m.Transactions = []*Transaction{}

	return &m
}

/**
 * Starts listeners and begins mining.
 */
func (m *Miner) initialize() {
	m.startNewSearch(nil)

	m.MClient.emitter.On(START_MINING, m.findProof)
	m.MClient.emitter.On(POST_TRANSACTION, m.addTransaction)
	m.MClient.emitter.Off(PROOF_FOUND, m.MClient.receiveBlock)
	m.MClient.emitter.On(PROOF_FOUND, m.receiveBlock)

	time.Sleep(0 * time.Second)
	m.MClient.emitter.Emit(START_MINING)

}

/**
 * Sets up the miner to start searching for a new block.
 *
 * @param {Set} [txSet] - Transactions the miner has that have not been accepted yet.
 */
func (m *Miner) startNewSearch(txSet map[*Transaction]int) {
	m.CurrentBlock = m.MClient.blockChain.makeBlock(m.MClient.address, m.MClient.lastBlock, nil, nil) //i,c set in makeBlock
	// Merging txSet into the transaction queue.
	// These transactions may include transactions not already included
	// by a recently received block, but that the miner is aware of.
	for tx, _ := range txSet {
		m.Transactions = append(m.Transactions, tx)
		m.addTransaction(tx)
	}

	// Add queued-up transactions to block.
	//for _, tx := range m.Transactions {
	//	m.CurrentBlock.addTransaction(tx, nil)
	//}
	m.Transactions = []*Transaction{} //clear

	// Start looking for a proof at 0.
	m.CurrentBlock.Proof = 0
}

/**
 * Looks for a "proof".  It breaks after some time to listen for messages.
 *
 * The 'oneAndDone' field is used for testing only; it prevents the findProof method
 * from looking for the proof again after the first attempt.
 *
 * @param {boolean} oneAndDone - Give up after the first PoW search (testing only).
 */
func (m *Miner) findProof() {
	pausePoint := m.CurrentBlock.Proof + m.MiningRounds
	//fmt.Printf("Current Block is (in findProof) %v \n", m.CurrentBlock.getId())
	//fmt.Printf("Current Block proof is %v , pausePoint is %v (in findProof)\n", m.CurrentBlock.Proof, pausePoint)
	for m.CurrentBlock.Proof < pausePoint {
		if m.CurrentBlock.hasValidProof() {
			fmt.Printf("%v found proof for block %v : %v \n", m.MClient.name, m.CurrentBlock.ChainLength, m.CurrentBlock.Proof)
			go m.announceProof()
			// Note: calling receiveBlock triggers a new search.
			//m.receiveBlock(m.CurrentBlock)
			m.startNewSearch(nil)
			break
		}
		m.CurrentBlock.Proof++
	}
	m.MClient.emitter.Emit(START_MINING)
}

/**
 * Broadcast the block, with a valid proof included.
 */
func (m *Miner) announceProof() {
	m.MClient.net.broadcast(PROOF_FOUND, m.CurrentBlock)
}

/**
 * Receives a block from another miner. If it is valid,
 * the block will be stored. If it is also a longer chain,
 * the miner will accept it and replace the currentBlock.
 *
 * @param {Block | Object} b - The block
 */
func (m *Miner) receiveBlock(b *Block) *Block {
	block := m.MClient.receiveBlock(b)
	if block == nil {
		errors.New("The block is invalid block")
		return nil
	}

	// We switch over to the new chain only if it is better.
	if m.CurrentBlock != nil && block.ChainLength >= m.CurrentBlock.ChainLength {
		fmt.Println("cutting over to new chain")
		txSet := m.syncTransactions(block)
		m.startNewSearch(txSet)
	}
	return block
}

/**
 * This function should determine what transactions
 * need to be added or deleted.  It should find a common ancestor (retrieving
 * any transactions from the rolled-back blocks), remove any transactions
 * already included in the newly accepted blocks, and add any remaining
 * transactions to the new block.
 *
 * @param {Block} nb - The newly accepted block.
 *
 * @returns {Set} - The set of transactions that have not yet been accepted by the new block.
 */
func (m *Miner) syncTransactions(nb *Block) map[*Transaction]int {
	cb := m.CurrentBlock
	cbTxs := make(map[*Transaction]int)
	nbTxs := make(map[*Transaction]int)

	// The new block may be ahead of the old block.  We roll back the new chain
	// to the matching height, collecting any transactions.
	for nb.ChainLength > cb.ChainLength {
		for _, tx := range nb.Transactions {
			nbTxs[tx] = 0 //add
			nb = m.MClient.blocks[string(nb.PrevBlockHash)]
		}
	}

	// Step back in sync until we hit the common ancestor.
	for cb != nil && nb != nil && cb.getId() != nb.getId() {
		// Store any transactions in the two chains.
		for _, tx := range cb.Transactions {
			cbTxs[tx] = 0
		}
		for _, tx := range nb.Transactions {
			nbTxs[tx] = 0
		}
		cb = m.MClient.blocks[string(cb.PrevBlockHash)]
		nb = m.MClient.blocks[string(nb.PrevBlockHash)]

	}

	// Remove all transactions that the new chain already has.
	//	fmt.Printf("BLOCK nb IN syncTransactions  (miner.go) - %v \n", nb.getId())
	//	fmt.Printf("LENGTH OF nb.Transactions (syncTransactions - miner.go) - %v \n", len(nb.Transactions))
	if len(nbTxs) != 0 { //keep getting error here
		for tx := range nbTxs {
			if _, ok := cbTxs[tx]; ok { //if currentBlocks has tx from new chain, delete it
				delete(cbTxs, tx)
			}
		}
	}
	return cbTxs
}

/**
 * Returns false if transaction is not accepted. Otherwise stores
 * the transaction to be added to the next block.
 *
 * @param {Transaction | String} tx - The transaction to add.
 */
func (m *Miner) addTransaction(tx *Transaction) bool {
	var addingtx *Transaction
	if tx == nil {
		addingtx = makeTransaction(tx) //need to fix makeTransaction in blockchain.go
	} else {
		addingtx = tx
	}
	m.Transactions = append(m.Transactions, addingtx)
	return m.CurrentBlock.addTransaction(addingtx, nil)
}

/**
 * When a miner posts a transaction, it must also add it to its current list of transactions.
 *
 * @param  {...any} args - Arguments needed for Client.postTransaction.
 */
func (m *Miner) postTransaction(outputs map[string]int, fee int) bool {
	//println("IN POST TRANSACTION miner.go")
	f := DEFAULT_TX_FEE
	if fee > DEFAULT_TX_FEE {
		f = fee
	}
	tx := m.MClient.postTransaction(outputs, f)
	return m.addTransaction(tx)
}
