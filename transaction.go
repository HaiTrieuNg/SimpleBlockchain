package main

import (
	"SpartanGold/utils"
	"crypto/rsa"
	"encoding/json"
)

type Transaction struct {
	from    string
	nonce   int
	pubKey  *rsa.PublicKey
	sig     []byte
	outputs map[string]int
	fee     int
	data    string
}

func NewTransaction(from string, nonce int, pubKey *rsa.PublicKey, sig []byte, fee int, outputs map[string]int, data string) *Transaction {
	tx := Transaction{from, nonce, pubKey, sig, outputs, fee, data}
	return &tx
}

/**
 * A transaction's ID is derived from its contents.
 */
func (t Transaction) getId() string {

	//create new obj with only needed properties
	//reference stringify -> Marshal https://go.dev/play/p/PWd9fpWrKZH

	var obj struct {
		from    string
		nonce   int
		pubKey  *rsa.PublicKey
		outputs map[string]int
		fee     int
		data    string
	}
	obj.from = t.from
	obj.nonce = t.nonce
	obj.pubKey = t.pubKey
	obj.outputs = t.outputs
	obj.fee = t.fee
	obj.data = t.data

	//jsonM := json.Marshal(&obj)

	out, err := json.Marshal(&obj)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("TRANSACTION ID: %v \n", string(utils.Hash("TX"+string(out))))
	return string(utils.Hash("TX" + string(out)))
}

/**
 * Signs a transaction and stores the signature in the transaction.
 *
 * @param privKey  - The key used to sign the signature.  It should match the
 *    public key included in the transaction.
 */
func (t *Transaction) sign(priveKey *rsa.PrivateKey) {
	t.sig = utils.Sign(priveKey, t.getId())
}

/**
 * Determines whether the signature of the transaction is valid
 * and if the from address matches the public key.
 *
 * @returns {Boolean} - Validity of the signature and from address.
 */
func (t Transaction) validSignature() bool {
	return t.sig != nil && utils.AddressMatchesKey(t.from, t.pubKey) && utils.VerifySignature(t.pubKey, t.getId(), t.sig)
}

/**
 * Verifies that there is currently sufficient gold for the transaction.
 *
 * @param {Block} block - Block used to check current balances
 *
 * @returns {boolean} - True if there are sufficient funds for the transaction,
 *    according to the balances from the specified block.
 */
func (t Transaction) sufficientFunds(block Block) bool {
	return t.totalOutput() <= block.Balances[t.from]
}

/**
 * Calculates the total value of all outputs, including the transaction fee.
 *
 * @returns {Number} - Total amount of gold given out with this transaction.
 */
func (t Transaction) totalOutput() int {
	total := 0
	for _, amount := range t.outputs {
		total += amount
	}
	//including the transaction fee.
	return total + t.fee
}
