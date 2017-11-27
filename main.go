package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var sentNotification map[string]Message

type NeoScanResponse struct {
	Transactions []NeoScanTransaction
}
type Vout struct {
	Value   float64 `json:"value"`
	N       int     `json:"n"`
	Asset   string  `json:"asset"`
	Address string  `json:"address"`
}

type NeoScanTransaction struct {
	Vouts []Vout `json:"vouts"`
	Vin   []struct {
		Value       float64 `json:"value"`
		Txid        string  `json:"txid"`
		N           int     `json:"n"`
		Asset       string  `json:"asset"`
		AddressHash string  `json:"address_hash"`
	} `json:"vin"`
	Version int    `json:"version"`
	Type    string `json:"type"`
	Txid    string `json:"txid"`
	Time    int    `json:"time"`
	SysFee  string `json:"sys_fee"`
	Size    int    `json:"size"`
	Scripts []struct {
		Verification string `json:"verification"`
		Invocation   string `json:"invocation"`
	} `json:"scripts"`
	Pubkey      interface{} `json:"pubkey"`
	Nonce       interface{} `json:"nonce"`
	NetFee      string      `json:"net_fee"`
	Description interface{} `json:"description"`
	Contract    interface{} `json:"contract"`
	Claims      []struct {
		Value       float64 `json:"value"`
		Txid        string  `json:"txid"`
		N           int     `json:"n"`
		Asset       string  `json:"asset"`
		AddressHash string  `json:"address_hash"`
	} `json:"claims"`
	BlockHeight int           `json:"block_height"`
	BlockHash   string        `json:"block_hash"`
	Attributes  []interface{} `json:"attributes"`
	Asset       interface{}   `json:"asset"`
}

func ContractTransaction(t NeoScanTransaction) {

	if len(t.Vouts) == 0 {
		return
	}
	receiver := Vout{}
	for _, vout := range t.Vouts {
		if vout.N == 0 {
			receiver = vout
		}
	}

	if len(t.Vin) == 0 {
		return
	}

	sender := t.Vin[0]

	//sending to oneself could mean that address is claiming gas
	if sender.AddressHash == receiver.Address {
		return
	}
	messageToReceiver := Message{
		Data: MessageData{
			Text: fmt.Sprintf("%v received %v %v from %v", receiver.Address, receiver.Value, receiver.Asset, sender.AddressHash),
		},
	}

	messageToSender := Message{
		Data: MessageData{
			Text: fmt.Sprintf("Your transaction of %v %v to %v has been proceesed", receiver.Value, receiver.Asset, receiver.Address),
		},
	}

	//send notification to receiver of asset
	Notify(t.Txid+receiver.Address, receiver.Address, messageToReceiver)
	//send notification to sender
	Notify(t.Txid+sender.AddressHash, sender.AddressHash, messageToSender)
}

func ClaimTransaction(t NeoScanTransaction) {
	if len(t.Vouts) == 0 {
		return
	}
	receiver := Vout{}
	for _, vout := range t.Vouts {
		if vout.N == 0 {
			receiver = vout
		}
	}

	message := Message{
		Data: MessageData{
			Text: fmt.Sprintf("%v claimed %f %v successfully", receiver.Address, receiver.Value, receiver.Asset),
		},
	}
	//send notification to address that claimed the gas
	Notify(t.Txid, receiver.Address, message)
}

func fetchTransaction(transactionType string) {
	log.Printf("fetch testnet %v", transactionType)
	//test net = https://neoscan-testnet.io/api/test_net/v1/get_last_transactions
	//main net = https://www.neoscan.io/api/main_net/v1/get_last_transactions
	url := "https://neoscan-testnet.io/api/test_net/v1/get_last_transactions/" + transactionType

	req, _ := http.NewRequest("GET", url, nil)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()

	response := NeoScanResponse{}

	err := json.NewDecoder(res.Body).Decode(&response.Transactions)
	if err != nil {
		log.Printf("error decode %v", err)
		return
	}

	//Vouts n = 0 is receiver
	//VIns n = 1 is sender
	for _, t := range response.Transactions {
		if t.Type == "ContractTransaction" {
			ContractTransaction(t)
		}
		if t.Type == "ClaimTransaction" {
			ClaimTransaction(t)
		}
	}
}

func fetchMainNetTransaction(transactionType string) {
	log.Printf("fetch mainnet %v", transactionType)
	//test net = https://neoscan-testnet.io/api/test_net/v1/get_last_transactions
	//main net =
	url := "https://www.neoscan.io/api/main_net/v1/get_last_transactions/" + transactionType

	req, _ := http.NewRequest("GET", url, nil)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()

	response := NeoScanResponse{}

	err := json.NewDecoder(res.Body).Decode(&response.Transactions)
	if err != nil {
		log.Printf("error decode %v", err)
		return
	}

	//Vouts n = 0 is receiver
	//VIns n = 1 is sender
	for _, t := range response.Transactions {
		if t.Type == "ContractTransaction" {
			ContractTransaction(t)
		}
		if t.Type == "ClaimTransaction" {
			ClaimTransaction(t)
		}
	}
}

func fetch(transactionType string, t time.Time) {
	log.Printf("\x1b[0m fetch %v transaction %+v", transactionType, t)
	go fetchTransaction(transactionType)
	go fetchMainNetTransaction(transactionType)
}

type MessageData struct {
	Text string `json:"text"`
}
type Message struct {
	Data MessageData `json:"data"`
}

func Notify(transactionID string, address string, message Message) {

	if (Message{}) != sentNotification[transactionID] {
		//log.Printf("already sent %v", transactionID)
		return
	}
	log.Printf("sending %v %v", message, transactionID)
	sentNotification[transactionID] = message

	//if not we don't make a request
	//AJs38kijktEuM22sjfXqfjZ734RqR4H6JW  o3 ios test wallet

	url := "https://api.getchannel.co/bot/publish/" + address

	payload, _ := json.Marshal(message)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))

	req.Header.Add("x-channel-application-key", "")
	req.Header.Add("x-channel-bot-token", "")
	req.Header.Add("content-type", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	// fmt.Println(res)
	fmt.Println(string(body))

	//we have to save if this transactionID is already notified
}

func doEvery(d time.Duration, f func(string, time.Time), transactionTypes ...string) {
	for x := range time.Tick(d) {
		for _, t := range transactionTypes {
			f(t, x)
		}

	}
}

func main() {
	sentNotification = map[string]Message{}
	fetch("ContractTransaction", time.Now())
	fetch("ClaimTransaction", time.Now())
	doEvery(2*time.Minute, fetch, "ContractTransaction", "ClaimTransaction")
}
