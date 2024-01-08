package chaincode

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

func (s *SmartContract) NewOrgStateId(stub shim.ChaincodeStubInterface, id string) (string, error) {
	return stub.CreateCompositeKey("Organization", []string{id})
}

func (s *SmartContract) NewOrgCreditStateId(stub shim.ChaincodeStubInterface, id string, orgId string) (string, error) {
	return stub.CreateCompositeKey("OrganizationCredit", []string{id, orgId})
}

func (s *SmartContract) NewOrgCreditLogStateId(stub shim.ChaincodeStubInterface, id string) (string, error) {
	return stub.CreateCompositeKey("OrganizationCreditLog", []string{id})
}
