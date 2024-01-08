/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"

	"github.com/diplom-mn/chaincode-go-organization/chaincode"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	diplomaChaincode, err := contractapi.NewChaincode(&chaincode.SmartContract{})
	if err != nil {
		log.Panicf("Error creating diploma chaincode: %v", err)
	}

	if err := diplomaChaincode.Start(); err != nil {
		log.Panicf("Error starting diploma chaincode: %v", err)
	}
}
