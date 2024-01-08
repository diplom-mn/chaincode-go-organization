package chaincode

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

func (s *SmartContract) newOrgStateId(stub shim.ChaincodeStubInterface, id string) (string, error) {
	return stub.CreateCompositeKey("Organization", []string{id})
}

func (s *SmartContract) newOrgCreditStateId(stub shim.ChaincodeStubInterface, id string, orgId string) (string, error) {
	return stub.CreateCompositeKey("OrganizationCredit", []string{id, orgId})
}

func (s *SmartContract) newOrgCreditLogStateId(stub shim.ChaincodeStubInterface, id string) (string, error) {
	return stub.CreateCompositeKey("OrganizationCreditLog", []string{id})
}
