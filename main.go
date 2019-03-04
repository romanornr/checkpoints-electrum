// Copyright (c) 2019 Romano (Viacoin developer)
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/romanornr/checkpoints-electrum/config"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"
)

const MINERCONFIRMATIONWINDOW uint64 = 2016

var c config.Conf

// Read RPC details first in config.yml file
func init() {
	c.GetConf()
}

// nBits is the target but stored in a compract format.
// this function converts the nBits to target again
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
		Host:         c.Host + ":" + c.RpcPort,
		User:         c.RpcUsername,
		Pass:         c.RpcPassword,
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
	Hash  string `json:"hash"`
	Time  uint64 `json:"time"`
	Nonce uint64 `json:"nonce"`
	Bits  uint32 `json:"bits,string"`
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

// write the checkpoints into a checkpoints.json file
func writeCheckpointsFile(list CheckpointList) {
	file, err := os.Create("checkpoints.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	jsonData, err := json.MarshalIndent(list, " ", "	")

	file.Write(jsonData)
}

// show how much blocks are left for the user
func ShowProgress(newestBlock uint64, currentBlock uint64) {
	blocksLeft := newestBlock - currentBlock
	fmt.Printf("%d / %d -> blocks left to parse: %d\n", newestBlock, currentBlock, blocksLeft)
}

func main() {

	start := time.Now()

	newestBlockResp, _ := client().GetBlockCount()
	newestBlock := uint64(newestBlockResp)

	var list CheckpointList

	var i uint64
	for i = MINERCONFIRMATIONWINDOW - 1; i < newestBlock; i += MINERCONFIRMATIONWINDOW {
		blockhashResp, _ := client().GetBlockHash(int64(i))
		blockhash, _ := json.Marshal(blockhashResp.String())

		resp, _ := client().RawRequest("getblock", []json.RawMessage{blockhash})

		if err := block.UnmarshalJSON(resp); err != nil {
			log.Fatal(err)
		}

		list = append(list, NewCheckpoint(block.Hash, CompactToBig(block.Bits), block.Time))
		ShowProgress(newestBlock, i)
	}

	writeCheckpointsFile(list)

	fmt.Printf("checkpoints.json has been saved !\n")
	log.Printf("Writing checkpoints took %s", time.Since(start))

	defer client().Shutdown()
}
