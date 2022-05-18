package main

import (
	"SpartanGold/utils"
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/chuckpreslar/emission"
)

type Client struct {
	net                         *fake_net
	name                        string
	keyPair                     *rsa.PrivateKey
	address                     string
	nonce                       int
	pendingOutgoingTransactions map[string]*Transaction
	pendingRecievedTransactions map[string]*Transaction
	blocks                      map[string]*Block
	blockChain                  *BlockChain
	pendingBlocks               map[string][]*Block
	lastBlock                   *Block
	lastConfirmedBlock          *Block
	receivedBlock               *Block
	emitter                     *emission.Emitter
}

type Message struct {
	from          string
	msg           string
	prevBlockHash string
	missingB      string
}

func NewClient(name string, net *fake_net, startingBlock *Block) *Client {
	var client Client

	client.net = net
	client.name = name

	client.keyPair = utils.GenerateKeypair()

	client.address = utils.CalcAddress(&client.keyPair.PublicKey)

	// Establishes order of transactions.  Incremented with each
	// new output transaction from this client.  This feature
	// avoids replay attacks.
	client.nonce = 0

	// A map of transactions where the client has spent money,
	// but where the transaction has not yet been confirmed.
	client.pendingOutgoingTransactions = make(map[string]*Transaction)
	// A map of transactions received but not yet confirmed.
	client.pendingRecievedTransactions = make(map[string]*Transaction)
	// A map of all block hashes to the accepted blocks.
	client.blocks = make(map[string]*Block)
	// A map of missing block IDS to the list of blocks depending
	// on the missing blocks.
	client.pendingBlocks = make(map[string][]*Block)

	if startingBlock != nil {
		client.setGenesisBlock(startingBlock)
	}

	client.net = net
	// Setting up listeners to receive messages from other clients.
	client.emitter = emission.NewEmitter()
	client.emitter.On(PROOF_FOUND, client.receiveBlock)
	client.emitter.On(MISSING_BLOCK, client.provideMissingBlock)

	return &client
}

func NewMsg(from string, msg string, prevH string, missingB string) Message {
	var mesg Message
	mesg.from = from
	mesg.msg = msg
	mesg.prevBlockHash = prevH
	mesg.missingB = missingB

	return mesg
}

/**
 * The genesis block can only be set if the client does not already
 * have the genesis block.
 *
 * @param startingBlock - The genesis block of the blockchain.
 */
func (client *Client) setGenesisBlock(startingBlock *Block) {
	if client.lastBlock != nil {
		fmt.Println("Cannot set genesis block for existing blockchain.")
	}

	client.lastConfirmedBlock = startingBlock
	client.lastBlock = startingBlock
	client.blocks[startingBlock.getId()] = startingBlock
}

/**
 * The amount of gold available to the client, not counting any pending
 * transactions.  This getter looks at the last confirmed block, since
 * transactions in newer blocks may roll back.
 */
func (client Client) getConfirmedBalance() int {
	return client.lastConfirmedBlock.balanceOf(client.address)
}

/**
 * Any gold received in the last confirmed block or before is considered
 * spendable, but any gold received more recently is not yet available.
 * However, any gold given by the client to other clients in unconfirmed
 * transactions is treated as unavailable.
 */
func (client Client) getAvailableGold() int {
	var pendingSpent int = 0
	for _, tx := range client.pendingOutgoingTransactions {
		pendingSpent += tx.totalOutput()
	}
	return client.getConfirmedBalance() - pendingSpent
}

/**
 * Broadcasts a transaction from the client giving gold to the clients
 * specified in 'outputs'. A transaction fee may be specified, which can
 * be more or less than the default value. (It's default fee for now)
 *
 * @param  outputs - The list of outputs of other addresses and
 *    amounts to pay.
 * @param  [fee] - The transaction fee reward to pay the miner.
 *
 * @returns Transaction - The posted transaction.
 */
func (client *Client) postTransaction(outputs map[string]int, fee int) *Transaction {

	f := DEFAULT_TX_FEE
	if fee > DEFAULT_TX_FEE {
		f = fee
	}
	totalPayments := f
	for _, amount := range outputs {
		totalPayments += amount
	}

	if totalPayments > client.getAvailableGold() {
		err := "Requested" + string(totalPayments) + ", but account only has" + string(client.getAvailableGold())
		errors.New(err)
	}

	return client.postGenericTransaction(outputs, f)
}

/**
 * Broadcasts a transaction from the client.  No validation is performed,
 * so the transaction might be rejected by other miners.
 *
 * This method is useful for handling special transactions with unique
 * parameters required, but generally should not be called directly by clients.
 *
 * @returns {Transaction} - The posted transaction.
 */

func (client *Client) postGenericTransaction(outputs map[string]int, fee int) *Transaction {
	tx := NewTransaction(client.address, client.nonce, &client.keyPair.PublicKey, nil, fee, outputs, "")
	tx.sign(client.keyPair)
	fmt.Printf("Transaction created %v - Signed? %v \n", tx.getId(), tx.validSignature())
	client.pendingOutgoingTransactions[tx.getId()] = tx
	client.nonce++
	fmt.Printf("NONCE %v \n", tx.getId(), client.nonce)
	client.net.broadcast(POST_TRANSACTION, tx)
	fmt.Printf("AFTER POST_TRANSACTION in postransaction client.go %v \n", tx.outputs)

	return tx
}

/**
 * Validates and adds a block to the list of blocks, possibly updating the head
 * of the blockchain.  Any transactions in the block are rerun in order to
 * update the gold balances for all clients.  If any transactions are found to be
 * invalid due to lack of funds, the block is rejected and 'null' is returned to
 * indicate failure.
 *
 * If any blocks cannot be connected to an existing block but seem otherwise valid,
 * they are added to a list of pending blocks and a request is sent out to get the
 * missing blocks from other clients.
 *
 * @param {Block | Object} block - The block to add to the clients list of available blocks.
 *
 * @returns {Block | null} The block with rerun transactions, or null for an invalid block.
 */
func (c *Client) receiveBlock(b *Block) *Block {
	block := b

	//if block is a string, deserialize (need to implement)

	//recieved previously
	if _, ok := c.blocks[block.getId()]; ok {
		//errors.New("Block was recieved previously")
		fmt.Printf("Block %v was recieved previously\n", string(block.getId()))
		return nil //just ignore, no error
	}

	//doesn't have valid proof
	if !block.hasValidProof() && !block.isGenesisBlock() {
		fmt.Printf("Block %v does not have a valid proof\n", string(block.getId()))
		return nil
	}

	// Make sure that we have the previous blocks, unless it is the genesis block.
	// If we don't have the previous blocks, request the missing blocks and exit.
	var prevBlock *Block = nil

	prevBlock, ok := c.blocks[string(block.PrevBlockHash)]
	if !ok && !block.isGenesisBlock() {
		stuckBlocks, ok := c.pendingBlocks[string(block.PrevBlockHash)]
		if !ok { //stuck block undefined
			c.requestMissingBlock(block)
			stuckBlocks = []*Block{}
		}
		stuckBlocks = append(stuckBlocks, block)
		c.pendingBlocks[string(block.PrevBlockHash)] = stuckBlocks
		return nil
	}

	if !block.isGenesisBlock() {
		success := block.rerun(prevBlock)
		if !success {
			return nil
		}
	}

	// Storing the block.
	c.blocks[block.getId()] = block

	// If it is a better block than the client currently has, set that
	// as the new currentBlock, and update the lastConfirmedBlock.
	if c.lastBlock.ChainLength < block.ChainLength {
		c.lastBlock = block
		c.setLastConfirmed()
	}

	// Go through any blocks that were waiting for this block
	// and recursively call receiveBlock.
	var unstuckBlocks []*Block
	if val, ok := c.pendingBlocks[block.getId()]; ok {
		unstuckBlocks = val
	}
	// Remove these blocks from the pending set.
	delete(c.pendingBlocks, block.getId())
	for _, ub := range unstuckBlocks {
		fmt.Printf("Processing unstuck block %v", block.getId())
		c.receiveBlock(ub)
	}
	return block
}

func (c *Client) receive(b *Block) {
	c.receiveBlock(b)
}

/**
 * Request the previous block from the network.
 *
 * @param {Block} block - The block that is connected to a missing block.
 */
func (client Client) requestMissingBlock(block *Block) {
	fmt.Printf("%v asks for missing block: %v", client, block.PrevBlockHash)
	var msg = Message{client.address, "", string(block.PrevBlockHash), ""}
	client.net.broadcast(MISSING_BLOCK, msg)
}

/**
 * Resend any transactions in the pending list.
 */
func (client Client) resendPendingTransactions() {
	for _, tx := range client.pendingOutgoingTransactions {
		client.net.broadcast(POST_TRANSACTION, tx)

	}
}

/**
 * Takes an object representing a request for a missing block.
 * If the client has the block, it will send the block to the
 * client that requested it.
 *
 * @param {Object} msg - Request for a missing block.
 * @param {String} msg.missing - ID of the missing block.
 */
func (client Client) provideMissingBlock(msg Message) {
	if msg.missingB != "" {
		if client.blocks[msg.missingB] != nil {
			fmt.Printf("Providing missing block %v", msg.missingB)
			mblock := client.blocks[msg.missingB]
			client.net.sendMessage(msg.from, PROOF_FOUND, mblock)
		}
	}
}

/**
 * Sets the last confirmed block according to the most recently accepted block,
 * also updating pending transactions according to this block.
 * Note that the genesis block is always considered to be confirmed.
 */
func (client *Client) setLastConfirmed() {
	block := client.lastBlock
	confirmedBlockHeight := block.ChainLength - CONFIRMED_DEPTH
	if confirmedBlockHeight < 0 {
		confirmedBlockHeight = 0
	}

	for block.ChainLength > confirmedBlockHeight {
		if _, ok := client.blocks[string(block.PrevBlockHash)]; ok {
			block = client.blocks[string(block.PrevBlockHash)]
		}
	}

	client.lastConfirmedBlock = block

	// Update pending transactions according to the new last confirmed block.
	for txID, tx := range client.pendingOutgoingTransactions {
		if client.lastConfirmedBlock.contains(tx) {
			delete(client.pendingOutgoingTransactions, txID)
		}
	}
}

/**
 * Utility method that displays all confirmed balances for all clients,
 * according to the client's own perspective of the network.
 */
func (client Client) showAllBalances() {

	fmt.Println("Showing balances:")
	for id, balance := range client.lastConfirmedBlock.Balances {
		fmt.Printf("%v has %v gold.\n", id, balance)
	}
}

/**
 * Logs messages to stdout, including the name to make debugging easier.
 * If the client does not have a name, then one is calculated from the
 * client's address.
 *
 * @param {String} msg - The message to display to the console.
 */
func (client Client) log(msg Message) {

	name := ""
	if client.name != "" {
		name = client.name
	} else {
		name = client.address[0:10]
	}
	fmt.Printf("%v log message: %v \n", name, msg.msg)
}

/**
 * Print out the blocks in the blockchain from the current head
 * to the genesis block.  Only the Block IDs are printed.
 */
func (client Client) showBlockchain() {
	block := client.lastBlock
	for block != nil {
		fmt.Println(block.getId())
		block = client.blocks[string(block.PrevBlockHash)]
	}
}
