package chaincode

import (
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func (s *SmartContract) IsDiplomMNClient(ctx contractapi.TransactionContextInterface) (bool, error) {
	role, roleFound, err := cid.GetAttributeValue(ctx.GetStub(), "diplom.mn.role")
	if err != nil {
		return false, err
	}
	if !roleFound {
		return false, fmt.Errorf("role attribute not found in identity - %s", err)
	}
	mspId, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return false, err
	}
	if mspId == "DsolutionsOrgMSP" && role == "diplom-mn-client" {
		return true, nil
	}
	return false, nil
}

func (s *SmartContract) IsIdentitySuperAdmin(ctx contractapi.TransactionContextInterface) error {
	err := cid.AssertAttributeValue(ctx.GetStub(), "diplom-mn.admin", "true")
	if err != nil {
		return InsufficientPermissionError
	}

	mspId, err := cid.GetMSPID(ctx.GetStub())
	if err != nil {
		return fmt.Errorf("Error when retrieving mspId")
	}
	if mspId != "DsolutionsOrgMSP" {
		return fmt.Errorf("Insufficient Permission on MSPID")
	}
	return nil
}

func (s *SmartContract) IdentityHasRoleOnOrg(ctx contractapi.TransactionContextInterface, orgId string, role string) error {
	err := cid.AssertAttributeValue(ctx.GetStub(), "diplom.mn.org.id", orgId)
	if err != nil {
		return fmt.Errorf("Insufficient Permission - orgId mismatch")
	}
	err = cid.AssertAttributeValue(ctx.GetStub(), "diplom.mn.org.role", role)
	if err != nil {
		return fmt.Errorf("Insufficient Role Permission")
	}
	return nil
}

func (s *SmartContract) IdentityHasOrgID(ctx contractapi.TransactionContextInterface, orgId string) error {
	err := cid.AssertAttributeValue(ctx.GetStub(), "diplom.mn.org.id", orgId)
	if err != nil {
		return fmt.Errorf("Insufficient Permission - orgId mismatch")
	}
	return nil
}

func (s *SmartContract) IsIdentitySuperAdminOrAdminOfOrg(ctx contractapi.TransactionContextInterface, orgId string) error {
	if s.IsIdentitySuperAdmin(ctx) != nil {
		if err := s.IdentityHasRoleOnOrg(ctx, orgId, "admin"); err != nil {
			return err
		}
	}
	return nil
}

func (s *SmartContract) IsIdentitySuperAdminOrHasAnyRoleOnOrg(ctx contractapi.TransactionContextInterface, orgId string) error {
	if s.IsIdentitySuperAdmin(ctx) != nil {
		if err := s.IdentityHasOrgID(ctx, orgId); err != nil {
			return err
		}
	}
	return nil
}
