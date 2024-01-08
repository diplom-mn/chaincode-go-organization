package chaincode_test

import (
	"testing"

	"github.com/diplom-mn/chaincode-org-go/chaincode"
	"github.com/diplom-mn/chaincode-org-go/chaincode/mocks"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/require"
)

//go:generate counterfeiter -o mocks/transaction.go -fake-name TransactionContext . transactionContext
type transactionContext interface {
	contractapi.TransactionContextInterface
}

//go:generate counterfeiter -o mocks/chaincodestub.go -fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

//go:generate counterfeiter -o mocks/statequeryiterator.go -fake-name StateQueryIterator . stateQueryIterator
type stateQueryIterator interface {
	shim.StateQueryIteratorInterface
}

func TestNewOrgStateId(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)
	contract := chaincode.SmartContract{}
	id, err := contract.NewOrgStateId(transactionContext.GetStub(), "ORG-MUST")
	require.Nil(t, err)
	require.Equal(t, "\x00ORG-MUST\x00", id)
}
