package main

import (
	"log"
	"github.com/btcsuite/btcd/rpcclient"
	"math/big"
	"fmt"
	"encoding/json"
	"strconv"
	"os"
	"io/ioutil"
)

func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}


func client() *rpcclient.Client {
	// Connect to local bitcoin/altcoin core RPC server using HTTP POST mode.
	connCfg := &rpcclient.ConnConfig{
		Host:         "127.0.0.1:5222",
		User:         "via",
		Pass:         "via",
		HTTPPostMode: true, // Viacoin core only supports HTTP POST mode
		DisableTLS:   true, // Viacoin core does not provide TLS by default
	}

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
		client.Shutdown()
	}
	//defer client.Shutdown()

	return client
}

type Request struct {
	Body string
}


type Block struct {
	Hash string `json:"hash"`
	Time uint64 `json:"time"`
	Nonce uint64 `json:"nonce"`
	Bits uint32 `json:"bits,string"`
}

type Hash struct {
	Hash string `json:"hash"`
}

var hash Hash
var block Block


func (block *Block) UnmarshalJSON(data []byte) error {
	type Alias Block

	aux := &struct {
		Bits string `json:"bits"`
		*Alias
	}{
		Alias: (*Alias)(block),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	// baseint is 16 because it's a hex
	i, err := strconv.ParseUint(aux.Bits, 16, 32)

	if err != nil {
		return err
	}

	block.Bits = uint32(i)

	return nil
}

type Checkpoint []interface{}
type CheckpointList []Checkpoint

func NewCheckpoint(hash string, target *big.Int, time uint64) Checkpoint {
	var checkpoint Checkpoint

	checkpoint = append(checkpoint, hash)
	checkpoint = append(checkpoint, target)
	checkpoint = append(checkpoint, time)

	return checkpoint

}

func writeCheckpointsFile(list CheckpointList){
	file, err := os.Create("checkpoints.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.Encode(&list)

	data, err := ioutil.ReadFile("checkpoints.json")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(data))
}

func main() {

	currentBlockResp, _:= client().GetBlockCount()
	currentBlock := int64(currentBlockResp)
	fmt.Println(currentBlock)

	var list CheckpointList

	var i int64
	for i = 2015; i < 5018214; i+=2016 {
		blockhashResp, _ := client().GetBlockHash(i)
		blockhash, _ := json.Marshal(blockhashResp.String())

		resp, _ := client().RawRequest("getblock", []json.RawMessage{blockhash})

		if err := block.UnmarshalJSON(resp); err != nil {
			log.Fatal(err)
		}

		list = append(list, NewCheckpoint(block.Hash, CompactToBig(block.Bits), block.Time))
		writeCheckpointsFile(list)

	}

	defer client().Shutdown()
}


