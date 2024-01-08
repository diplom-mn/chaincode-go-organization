package chaincode

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type Organization struct {
	DocType           string `json:"docType"`
	ID                string `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	InstitutionID     string `json:"institutionId"`
	InstitutionName   string `json:"institutionName"`
	Desc              string `json:"desc"`
	OrgCreditID       string `json:"orgCreditId"`
	LogoUrl           string `json:"logoUrl"`
	IsActive          bool   `json:"isActive"`
	PubKeyType        string `json:"pubKeyType"`
	PubKeyPem         string `json:"pubKeyPem"`
	CreateTxTimestamp int64  `json:"createTxTimestamp"`
	UpdateTxTimestamp int64  `json:"updateTxTimestamp"`
}

func (s *SmartContract) OrgExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	orgId, err := s.NewOrgStateId(ctx.GetStub(), id)
	if err != nil {
		return false, fmt.Errorf("Org ID error - %s", err)
	}
	org, err := ctx.GetStub().GetState(orgId)
	if err != nil {
		return false, fmt.Errorf("Failed to retrieve org - %s", err)
	}
	return org != nil, nil
}

func (s *SmartContract) CreateOrg(ctx contractapi.TransactionContextInterface, orgId string, name string, desc string, institutionId string, institutionName, logo string, initialCredit string, creditDesc string, isActive bool) error {
	err := s.IsIdentitySuperAdmin(ctx)
	if err != nil {
		return err
	}
	orgExists, err := s.OrgExists(ctx, orgId)
	if err != nil {
		return err
	}
	if orgExists {
		return fmt.Errorf("Org %s already exists", orgId)
	}
	orgStateId, err := s.NewOrgStateId(ctx.GetStub(), orgId)
	if err != nil {
		return err
	}
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	orgCredit, err := s.CreateCredit(ctx, orgId, desc, initialCredit)
	if err != nil {
		return err
	}
	org := Organization{
		DocType:           "Organization",
		ID:                orgId,
		Name:              name,
		Desc:              desc,
		InstitutionID:     institutionId,
		InstitutionName:   institutionName,
		OrgCreditID:       orgCredit.ID,
		LogoUrl:           logo,
		IsActive:          isActive,
		PubKeyType:        "",
		PubKeyPem:         "",
		CreateTxTimestamp: ts.AsTime().UTC().Unix(),
	}
	orgJSON, err := json.Marshal(org)
	if err != nil {
		return err
	}
	if err = ctx.GetStub().PutState(orgStateId, orgJSON); err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) UpdateOrg(ctx contractapi.TransactionContextInterface, orgId string, name string, desc string, email string, institutionId string, institutionName, logo string, initialCredit string, creditDesc string, isActive bool, pubKeyType string, pubKeyPem string) error {
	err := s.IsIdentitySuperAdmin(ctx)
	if err != nil {
		return err
	}
	orgExists, err := s.OrgExists(ctx, orgId)
	if err != nil {
		return err
	}
	if orgExists {
		return fmt.Errorf("Org %s already exists", orgId)
	}
	orgStateId, err := s.NewOrgStateId(ctx.GetStub(), orgId)
	if err != nil {
		return err
	}
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return err
	}
	org, err := s.ReadOrg(ctx, orgId)
	if err != nil {
		return err
	}
	org.Name = name
	org.Desc = desc
	org.Email = email
	org.InstitutionID = institutionId
	org.InstitutionName = institutionName
	org.InstitutionID = institutionId
	org.InstitutionID = institutionId
	org.LogoUrl = logo
	org.IsActive = isActive
	org.UpdateTxTimestamp = ts.AsTime().UTC().Unix()
	orgJSON, err := json.Marshal(org)
	if err != nil {
		return err
	}
	if err = ctx.GetStub().PutState(orgStateId, orgJSON); err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) SetOrgPublicKey(ctx contractapi.TransactionContextInterface, id string, pubKeyType string, pubKeyPemArg string) error {
	if err := s.IsIdentitySuperAdmin(ctx); err != nil {
		return err
	}
	if pubKeyType != "ecdsa:P-384" {
		return fmt.Errorf("Unsupported Pub key type")
	}
	pemBlock, _ := pem.Decode([]byte(pubKeyPemArg))
	pubKey, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return err
	}
	_, pubKeyOk := pubKey.(*ecdsa.PublicKey)
	if !pubKeyOk {
		return fmt.Errorf("Public key invalid")
	}
	stateId, err := s.NewOrgStateId(ctx.GetStub(), id)
	if err != nil {
		return err
	}
	orgJSON, err := ctx.GetStub().GetState(stateId)
	if err != nil {
		return err
	}
	var org Organization
	err = json.Unmarshal(orgJSON, &org)
	if err != nil {
		return err
	}

	if len(org.PubKeyPem) > 0 {
		return fmt.Errorf("Org already has a public key")
	}

	pubKeyQueryString := strings.ReplaceAll(pubKeyPemArg, "\n", "\\n")
	queryString := fmt.Sprintf(`
	{
		"selector": {
			"pubKeyPem":"%s"
		}
	}`, pubKeyQueryString)

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return err
	}
	defer resultsIterator.Close()
	if resultsIterator.HasNext() {
		return fmt.Errorf("Public key is already taken")
	}

	org.PubKeyType = pubKeyType
	org.PubKeyPem = pubKeyPemArg
	updatedJSON, err := json.Marshal(org)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(stateId, updatedJSON); err != nil {
		return err
	}

	return nil
}

func (s *SmartContract) ReadOrg(ctx contractapi.TransactionContextInterface, id string) (*Organization, error) {
	return s.readOrg(ctx.GetStub(), id)
}

func (s *SmartContract) readOrg(stub shim.ChaincodeStubInterface, id string) (*Organization, error) {
	stateId, err := s.NewOrgStateId(stub, id)
	if err != nil {
		return nil, err
	}
	orgJSON, err := stub.GetState(stateId)
	if err != nil {
		return nil, err
	}
	var org Organization
	err = json.Unmarshal(orgJSON, &org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (s *SmartContract) ReadMyOrg(ctx contractapi.TransactionContextInterface) (*Organization, error) {

	orgId, orgIdFound, err := cid.GetAttributeValue(ctx.GetStub(), "diplom.mn.org.id")
	if !orgIdFound {
		return nil, fmt.Errorf("diplom.mn.org.id not found in idendity")
	}

	org, err := s.ReadOrg(ctx, orgId)
	if err != nil {
		return nil, err
	}
	return org, nil
}

func (s *SmartContract) readMyOrg(stub shim.ChaincodeStubInterface) (*Organization, error) {

	orgId, orgIdFound, err := cid.GetAttributeValue(stub, "diplom.mn.org.id")
	if !orgIdFound {
		return nil, fmt.Errorf("diplom.mn.org.id not found in idendity")
	}

	org, err := s.readOrg(stub, orgId)
	if err != nil {
		return nil, err
	}
	return org, nil
}

func (t *SmartContract) ListOrgs(ctx contractapi.TransactionContextInterface) ([]*Organization, error) {
	queryString := fmt.Sprintf(`
	{
		"selector": {
			"docType":"%s"
		},
		"sort": [
			{
				"createTxTimestamp": "asc"
			}
		]
	}`, "Organization")
	return getQueryResultForQueryStringFromOrg(ctx, queryString)
}

func getQueryResultForQueryStringFromOrg(ctx contractapi.TransactionContextInterface, queryString string) ([]*Organization, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	return constructQueryResponseFromIteratorFromOrg(resultsIterator)
}

func constructQueryResponseFromIteratorFromOrg(resultsIterator shim.StateQueryIteratorInterface) ([]*Organization, error) {
	var orgs []*Organization = make([]*Organization, 0)
	for resultsIterator.HasNext() {
		queryResult, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var org Organization
		err = json.Unmarshal(queryResult.Value, &org)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, &org)
	}

	return orgs, nil
}
