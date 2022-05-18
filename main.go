package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Starting simulation.  This may take a moment...")

	fakeNet := NewFakeNet()

	//Clients
	Alice := NewClient("Alice", fakeNet, nil)
	Bob := NewClient("Bob", fakeNet, nil)
	Charlie := NewClient("Charlie", fakeNet, nil)
	//Miners
	Minnie := NewMiner("Minnie", fakeNet, nil)
	Mickey := NewMiner("Mickey", fakeNet, nil)

	// Creating genesis block
	bc := BlockChain{}

	clientBalanceMap := make(map[*Client]int)
	clientBalanceMap[Alice] = 233
	clientBalanceMap[Bob] = 99
	clientBalanceMap[Charlie] = 67
	clientBalanceMap[Minnie.MClient] = 400
	clientBalanceMap[Mickey.MClient] = 300

	g := bc.makeGenesis(clientBalanceMap, nil)
	fmt.Println("Serialize: %v", g.serialize())

	showBalances := func(client Client) {
		/*fmt.Printf("Alice has  %v gold.\n", Alice.showAllBalances)
		fmt.Printf("Bob has  %v gold.\n", Bob.showAllBalances)
		fmt.Printf("Charlie has  %v gold.\n", Charlie.showAllBalances)
		fmt.Printf("Minnie has  %v gold.\n", Minnie.MClient.showAllBalances)
		fmt.Printf("Mickey has %v gold.\n", Mickey.MClient.showAllBalances)*/
		//fmt.Printf("Last confirmed block Id: %v \n", client.lastBlock.getId())
		fmt.Printf("Alice has  %v gold.\n", client.lastBlock.balanceOf(Alice.address))
		fmt.Printf("Bob has  %v gold.\n", client.lastBlock.balanceOf(Bob.address))
		fmt.Printf("Charlie has  %v gold.\n", client.lastBlock.balanceOf(Charlie.address))
		fmt.Printf("Minnie has  %v gold.\n", client.lastBlock.balanceOf(Minnie.MClient.address))
		fmt.Printf("Mickey has %v gold.\n", client.lastBlock.balanceOf(Mickey.MClient.address))

	}

	// Showing the initial balances from Alice's perspective, for no particular reason.
	fmt.Println("Initial balances:")
	showBalances(*Alice)
	//Alice.showAllBalances();
	clientList := []*Client{Alice, Bob, Charlie, Minnie.MClient, Mickey.MClient}
	fakeNet.register(clientList)

	// Miners start mining.
	go Minnie.initialize()
	go Mickey.initialize()

	// Alice transfers some money to Bob.
	// Alice aso transfer some money to Charlie
	fmt.Printf("Alice is transfering 40 gold to Bob- %v and 30 gold to Charlie-%v.\n", Bob.address, Charlie.address)
	Alice.postTransaction(map[string]int{Bob.address: 40, Charlie.address: 30}, 3)
	time.Sleep(7 * time.Second)
	fmt.Println()
	fmt.Printf("Minnie has a chain of length %v:", Minnie.MClient.lastBlock.ChainLength)

	fmt.Println()
	fmt.Printf("Mickey has a chain of length %v:", Mickey.MClient.lastBlock.ChainLength)

	fmt.Println()
	fmt.Println("Final balances (Minnie's perspective):")
	showBalances(*Minnie.MClient)
	//Minnie.MClient.showAllBalances();

	fmt.Println()
	fmt.Println("Final balances (Mickey's perspective):")
	showBalances(*Mickey.MClient)
	//Minnie.MClient.showAllBalances();

	fmt.Println()
	fmt.Println("Final balances (Alice's perspective):")
	showBalances(*Alice)
}
