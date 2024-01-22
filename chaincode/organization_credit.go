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
	TxID        string `json:"txID"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	OrgID       string `json:"orgId"`
	CreditID    string `json:"creditId"`
	Amount      string `json:"amount"`
	Credit      string `json:"credit"`
	Debit       string `json:"debit"`
	TxTimestamp int64  `json:"txTimestamp"`
}

type ListOrgCreditLog struct {
	BookMark string          `json:"bookMark" validate:"required"`
	Records  []*OrgCreditLog `json:"records" validate:"required"`
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
	err = createCreditLog(s, ctx.GetStub(), orgCredit, "Create Credit", "mint", "0", amount)
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

	if err = s.mintOrgCredit(ctx, creditId, orgId, amount, title, ts.AsTime().Unix(), "mint"); err != nil {
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
	if err = s.burnOrgCredit(ctx.GetStub(), creditId, orgId, amount, title, ts.AsTime().Unix(), "burn"); err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) SpendCredit(ctx contractapi.TransactionContextInterface, creditId string, orgId string, amount string, title string) error {
	if err := s.IsIdentityAdminOfOrg(ctx, orgId); err != nil {
		return err
	}
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	if err = s.burnOrgCredit(ctx.GetStub(), creditId, orgId, amount, title, ts.AsTime().Unix(), "spend"); err != nil {
		return err
	}
	return nil
}

// mints credit and create log without checking any permission
func (s *SmartContract) mintOrgCredit(ctx contractapi.TransactionContextInterface, creditId string, orgId string, amount string, title string, ts int64, logType string) error {
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
	if err = createCreditLog(s, ctx.GetStub(), orgCredit, title, logType, "0", amount); err != nil {
		return err
	}
	return nil
}

// burns credit and create log without checking any permission
func (s *SmartContract) burnOrgCredit(stub shim.ChaincodeStubInterface, creditId string, orgId string, amount string, title string, ts int64, logType string) error {
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
	if err = createCreditLog(s, stub, orgCredit, title, logType, amount, "0"); err != nil {
		return err
	}
	return nil
}

func createCreditLog(s *SmartContract, stub shim.ChaincodeStubInterface, orgCredit OrgCredit, title string, logType string, credit string, debit string) error {
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
		TxID:        stub.GetTxID(),
		ID:          id,
		CreditID:    orgCredit.ID,
		OrgID:       orgCredit.OrgID,
		Type:        logType,
		Title:       title,
		Amount:      orgCredit.Amount,
		Credit:      credit,
		Debit:       debit,
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
				"txTimestamp": "%s"
			}
		]
	}`, "OrgCreditLog", credit.ID, credit.OrgID, "desc")
	parsed, _, err := getQueryResultForQueryStringFromCreditLog(ctx, queryString, 100, "")
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func (s *SmartContract) ListCreditLogPaginated(ctx contractapi.TransactionContextInterface, orgId string, creditId string, sortArg string, pageSize int32, bookMark string) (*ListOrgCreditLog, error) {
	credit, err := s.ReadCredit(ctx, creditId, orgId)
	if err != nil {
		return nil, err
	}
	if sortArg != "asc" && sortArg != "desc" {
		return nil, fmt.Errorf("sortArg should be either of asc org desc")
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
				"txTimestamp": "%s"
			}
		]
	}`, "OrgCreditLog", credit.ID, credit.OrgID, sortArg)
	parsed, bookMark, err := getQueryResultForQueryStringFromCreditLog(ctx, queryString, pageSize, bookMark)
	if err != nil {
		return nil, err
	}
	return &ListOrgCreditLog{
		BookMark: bookMark,
		Records:  parsed,
	}, nil
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

func getQueryResultForQueryStringFromCreditLog(ctx contractapi.TransactionContextInterface, queryString string, pageSize int32, bookMark string) ([]*OrgCreditLog, string, error) {
	resultsIterator, meta, err := ctx.GetStub().GetQueryResultWithPagination(queryString, pageSize, bookMark)
	if err != nil {
		return nil, "", err
	}
	defer resultsIterator.Close()

	parsed, err := constructQueryResponseFromIteratorFromCreditLog(resultsIterator)
	if err != nil {
		return nil, "", err
	}
	return parsed, meta.Bookmark, err
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
