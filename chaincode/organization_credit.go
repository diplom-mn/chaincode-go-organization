package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/shopspring/decimal"
)

type OrgCredit struct {
	DocType     string `json:"docType"`
	ID          string `json:"id"`
	OrgID       string `json:"orgId"`
	Amount      string `json:"amount"`
	TxTimestamp int64  `json:"txTimestamp"`
}

type OrgCreditLog struct {
	DocType     string `json:"docType"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	OrgID       string `json:"orgId"`
	CreditID    string `json:"creditId"`
	Amount      string `json:"amount"`
	TxTimestamp int64  `json:"txTimestamp"`
}

func (s *SmartContract) CreditExists(ctx contractapi.TransactionContextInterface, id string, orgId string) (bool, error) {
	if err := s.IsIdentitySuperAdmin(ctx); err != nil {
		return false, err
	}
	stateId, err := s.newOrgCreditStateId(ctx.GetStub(), id, orgId)
	if err != nil {
		return false, err
	}
	state, err := ctx.GetStub().GetState(stateId)
	if err != nil {
		return false, fmt.Errorf("Failed to read world state - %s", err)
	}
	return state != nil, nil
}

func (s *SmartContract) CreateCredit(ctx contractapi.TransactionContextInterface, orgId string, title string, amount string) (*OrgCredit, error) {
	if err := s.IsIdentitySuperAdmin(ctx); err != nil {
		return nil, err
	}
	creditId := orgId
	exists, err := s.CreditExists(ctx, creditId, orgId)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("Credit %s already exists", creditId)
	}
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, err
	}
	stateId, err := s.newOrgCreditStateId(ctx.GetStub(), creditId, orgId)
	if err != nil {
		return nil, err
	}
	orgCredit := OrgCredit{
		DocType:     "OrgCredit",
		ID:          creditId,
		OrgID:       orgId,
		Amount:      amount,
		TxTimestamp: ts.AsTime().Unix(),
	}
	orgCreditJSON, err := json.Marshal(orgCredit)
	if err != nil {
		return nil, err
	}
	err = ctx.GetStub().PutState(stateId, orgCreditJSON)
	err = createCreditLog(s, ctx.GetStub(), orgCredit, "Create Credit")
	return &orgCredit, err
}

func (s *SmartContract) MintCredit(ctx contractapi.TransactionContextInterface, creditId string, orgId string, amount string, title string) error {
	if err := s.IsIdentitySuperAdmin(ctx); err != nil {
		return err
	}
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}

	if err = s.mintOrgCredit(ctx, creditId, orgId, amount, title, ts.AsTime().Unix()); err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) BurnCredit(ctx contractapi.TransactionContextInterface, creditId string, orgId string, amount string, title string) error {
	if err := s.IsIdentitySuperAdmin(ctx); err != nil {
		return err
	}
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	if err = s.burnOrgCredit(ctx.GetStub(), creditId, orgId, amount, title, ts.AsTime().Unix()); err != nil {
		return err
	}
	return nil
}

// mints credit and create log without checking any permission
func (s *SmartContract) mintOrgCredit(ctx contractapi.TransactionContextInterface, creditId string, orgId string, amount string, title string, ts int64) error {
	creditStateId, err := s.newOrgCreditStateId(ctx.GetStub(), creditId, orgId)
	if err != nil {
		return err
	}
	orgCreditJSON, err := ctx.GetStub().GetState(creditStateId)
	if err != nil {
		return err
	}
	var orgCredit OrgCredit
	if err = json.Unmarshal(orgCreditJSON, &orgCredit); err != nil {
		return err
	}
	creditAmount, err := decimal.NewFromString(amount)
	if creditAmount.LessThan(decimal.Zero) {
		return fmt.Errorf("Credit is lower than 0")
	}
	if err != nil {
		return err
	}
	oldCreditAmount, err := decimal.NewFromString(orgCredit.Amount)
	if err != nil {
		return err
	}

	newAmount := creditAmount.Add(oldCreditAmount)

	orgCredit.Amount = newAmount.String()
	orgCredit.TxTimestamp = ts

	newOrgCreditJSON, err := json.Marshal(orgCredit)
	if err != nil {
		return err
	}

	if err = ctx.GetStub().PutState(creditStateId, newOrgCreditJSON); err != nil {
		return err
	}
	if err = createCreditLog(s, ctx.GetStub(), orgCredit, title); err != nil {
		return err
	}
	return nil
}

// burns credit and create log without checking any permission
func (s *SmartContract) burnOrgCredit(stub shim.ChaincodeStubInterface, creditId string, orgId string, amount string, title string, ts int64) error {
	creditStateId, err := s.newOrgCreditStateId(stub, creditId, orgId)
	if err != nil {
		return err
	}
	orgCreditJSON, err := stub.GetState(creditStateId)
	if err != nil {
		return err
	}
	var orgCredit OrgCredit
	err = json.Unmarshal(orgCreditJSON, &orgCredit)
	if err != nil {
		return err
	}

	oldCreditAmount, err := decimal.NewFromString(orgCredit.Amount)
	if err != nil {
		return err
	}

	subtractAmount, err := decimal.NewFromString(amount)
	if subtractAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("Credit is lower than or equals to 0")
	}
	if subtractAmount.GreaterThan(oldCreditAmount) {
		return fmt.Errorf("Amount exceeds remaining credit")
	}

	newAmount := oldCreditAmount.Sub(subtractAmount)

	orgCredit.Amount = newAmount.String()
	orgCredit.TxTimestamp = ts

	newOrgCreditJSON, err := json.Marshal(orgCredit)
	if err != nil {
		return err
	}
	if err = stub.PutState(creditStateId, newOrgCreditJSON); err != nil {
		return err
	}
	if err = createCreditLog(s, stub, orgCredit, title); err != nil {
		return err
	}
	return nil
}

func createCreditLog(s *SmartContract, stub shim.ChaincodeStubInterface, orgCredit OrgCredit, title string) error {
	id := stub.GetTxID()
	orgCreditLogStateId, err := s.newOrgCreditLogStateId(stub, id)
	if err != nil {
		return err
	}
	existing, err := stub.GetState(orgCreditLogStateId)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("Credit %s already exists", id)
	}
	ts, err := stub.GetTxTimestamp()
	if err != nil {
		return err
	}
	orgCreditLog := OrgCreditLog{
		DocType:     "OrgCreditLog",
		ID:          id,
		CreditID:    orgCredit.ID,
		OrgID:       orgCredit.OrgID,
		Title:       title,
		Amount:      orgCredit.Amount,
		TxTimestamp: ts.AsTime().Unix(),
	}
	orgCreditLogJSON, err := json.Marshal(orgCreditLog)
	if err != nil {
		return err
	}
	if err = stub.PutState(orgCreditLogStateId, orgCreditLogJSON); err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) ListCreditLog(ctx contractapi.TransactionContextInterface, orgId string, creditId string) ([]*OrgCreditLog, error) {
	credit, err := s.ReadCredit(ctx, creditId, orgId)
	if err != nil {
		return nil, err
	}
	queryString := fmt.Sprintf(`
	{
		"selector": {
			"docType":"%s", 
			"creditId": "%s",
			"orgId": "%s"
		},
		"sort": [
			{
				"txTimestamp": "desc"
			}
		]
	}`, "OrgCreditLog", credit.ID, credit.OrgID)
	return getQueryResultForQueryStringFromCreditLog(ctx, queryString)
}

func (s *SmartContract) ReadCredit(ctx contractapi.TransactionContextInterface, creditId string, orgId string) (*OrgCredit, error) {
	if err := s.IsIdentitySuperAdminOrAdminOfOrg(ctx, orgId); err != nil {
		return nil, err
	}
	creditStateId, err := s.newOrgCreditStateId(ctx.GetStub(), creditId, orgId)
	if err != nil {
		return nil, err
	}
	creditJSON, err := ctx.GetStub().GetState(creditStateId)
	var credit OrgCredit
	if err = json.Unmarshal(creditJSON, &credit); err != nil {
		return nil, err
	}
	return &credit, err
}

func getQueryResultForQueryStringFromCreditLog(ctx contractapi.TransactionContextInterface, queryString string) ([]*OrgCreditLog, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIteratorFromCreditLog(resultsIterator)
}

func constructQueryResponseFromIteratorFromCreditLog(resultsIterator shim.StateQueryIteratorInterface) ([]*OrgCreditLog, error) {
	var data []*OrgCreditLog = make([]*OrgCreditLog, 0)
	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var item OrgCreditLog
		err = json.Unmarshal(queryResult.Value, &item)
		if err != nil {
			return nil, err
		}
		data = append(data, &item)
	}

	return data, nil
}
