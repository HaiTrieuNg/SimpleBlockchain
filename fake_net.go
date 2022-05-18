package main

type fake_net struct {
	Clients map[string]*Client
}

func NewFakeNet() *fake_net {
	var f fake_net
	f.Clients = make(map[string]*Client)
	return &f
}

/**
 * Registers clients to the network.
 * Clients and Miners are registered by public key.
 *
 * @param {...Object} clientList - clients to be registered to this network (may be Client or Miner)
 */
func (f *fake_net) register(clientList []*Client) {
	for _, client := range clientList {
		f.Clients[client.address] = client
	}

}

/**
 * Broadcasts to all clients within this.clients the message msg and payload o.
 *
 * @param {String} msg - the name of the event being broadcasted (e.g. "PROOF_FOUND")
 * @param {Object} o - payload of the message
 */
func (f *fake_net) broadcast(msg string, o interface{}) {
	for address, _ := range f.Clients {
		f.sendMessage(address, msg, o)
	}
}

/**
 * Sends message msg and payload o directly to Client name.
 *
 * The message may be lost or delayed, with the probability
 * defined for this instance.
 *
 * @param {String} address - the public key address of the client or miner to which to send the message
 * @param {String} msg - the name of the event being broadcasted (e.g. "PROOF_FOUND")
 * @param {Object} o - payload of the message
 */
func (f *fake_net) sendMessage(address string, msg string, jsonObj interface{}) {
	// Serializing/deserializing the object to prevent cheating in single threaded mode.
	//...
	client := f.Clients[address]
	client.emitter.Emit(msg, jsonObj)
}

/**
 * Tests whether a client is registered with the network.
 *
 * @param {Client} client - the client to test for.
 *
 * @returns {boolean} True if the client is already registered.
 */
func (f *fake_net) recognizes(client Client) bool {
	if _, ok := f.Clients[client.address]; ok {
		return true
	} else {
		return false
	}
}
