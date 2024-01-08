package chaincode

import (
	"strconv"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type Ctx struct {
	contractapi.TransactionContextInterface
	stub shim.ChaincodeStubInterface
}

func (s *SmartContract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	if fn == "burnOrgCredit" {
		ts, err := strconv.ParseInt(args[4], 10, 64)
		if err != nil {
			return shim.Error(err.Error())
		}
		org, err := s.readMyOrg(stub)
		if err != nil {
			return shim.Error(err.Error())
		}
		if org.ID != args[1] {
			return shim.Error(InsufficientPermissionError.Error())
		}
		err = s.burnOrgCredit(stub, args[0], args[1], args[2], args[3], ts)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	return shim.Success(nil)
}
