package accountv2

import (
	"fmt"

	"github.com/IBM-Cloud/bluemix-go/bmxerror"
	"github.com/IBM-Cloud/bluemix-go/client"
)

//Metadata ...
type Metadata struct {
	GUID string `json:"guid"`
	URL  string `json:"url"`
}

//Resource ...
type Resource struct {
	Metadata Metadata
}

//Account Model ...
type Account struct {
	GUID          string
	Name          string
	Type          string
	State         string
	OwnerGUID     string
	OwnerUserID   string
	OwnerUniqueID string
	CustomerID    string
	CountryCode   string
	CurrencyCode  string
	Organizations []AccountOrganization
	Members       []AccountMember `json:"members"`
}

//AccountOrganization ...
type AccountOrganization struct {
	GUID   string `json:"guid"`
	Region string `json:"region"`
}

//AccountMember ...
type AccountMember struct {
	GUID     string `json:"guid"`
	UserID   string `json:"user_id"`
	UniqueID string `json:"unique_id"`
}

//AccountResource ...
type AccountResource struct {
	Resource
	Entity AccountEntity
}

//AccountEntity ...
type AccountEntity struct {
	Name          string                `json:"name"`
	Type          string                `json:"type"`
	State         string                `json:"state"`
	OwnerGUID     string                `json:"owner"`
	OwnerUserID   string                `json:"owner_userid"`
	OwnerUniqueID string                `json:"owner_unique_id"`
	CustomerID    string                `json:"customer_id"`
	CountryCode   string                `json:"country_code"`
	CurrencyCode  string                `json:"currency_code"`
	Organizations []AccountOrganization `json:"organizations_region"`
	Members       []AccountMember       `json:"members"`
}

//ToModel ...
func (resource AccountResource) ToModel() Account {
	entity := resource.Entity

	return Account{
		GUID:          resource.Metadata.GUID,
		Name:          entity.Name,
		Type:          entity.Type,
		State:         entity.State,
		OwnerGUID:     entity.OwnerGUID,
		OwnerUserID:   entity.OwnerUserID,
		OwnerUniqueID: entity.OwnerUniqueID,
		CustomerID:    entity.CustomerID,
		CountryCode:   entity.CountryCode,
		CurrencyCode:  entity.CurrencyCode,
		Organizations: entity.Organizations,
		Members:       entity.Members,
	}
}

func (nameQueryResponse AccountNameQueryResponse) ToModel() Account {
	entity := nameQueryResponse.Entity
	guid := nameQueryResponse.Metadata.GUID

	return Account{
		GUID:          guid,
		Name:          entity.Name,
		Type:          entity.Type,
		State:         entity.State,
		OwnerGUID:     entity.OwnerGUID,
		OwnerUserID:   entity.OwnerUserID,
		OwnerUniqueID: entity.OwnerUniqueID,
		CustomerID:    entity.CustomerID,
		CountryCode:   entity.CountryCode,
		CurrencyCode:  entity.CurrencyCode,
		Organizations: entity.Organizations,
		Members:       entity.Members,
	}
}

//AccountQueryResponse ...
type AccountQueryResponse struct {
	Metadata Metadata
	Accounts []AccountResource `json:"resources"`
}

//AccountQueryResponse ...
type AccountNameQueryResponse struct {
	Metadata Metadata
	Entity   AccountEntity
}

//Accounts ...
type Accounts interface {
	List() ([]Account, error)
	FindByOrg(orgGUID string, region string) (*Account, error)
	FindByOwner(userID string) (*Account, error)
	Get(accountId string) (*Account, error)
}

type account struct {
	client *client.Client
}

func newAccountAPI(c *client.Client) Accounts {
	return &account{
		client: c,
	}
}

//FindByOrg ...
func (a *account) FindByOrg(orgGUID, region string) (*Account, error) {
	type organizationRegion struct {
		GUID   string `json:"guid"`
		Region string `json:"region"`
	}

	payLoad := struct {
		OrganizationsRegion []organizationRegion `json:"organizations_region"`
	}{
		OrganizationsRegion: []organizationRegion{
			{
				GUID:   orgGUID,
				Region: region,
			},
		},
	}

	queryResp := AccountQueryResponse{}
	response, err := a.client.Post("/coe/v2/getaccounts", payLoad, &queryResp)
	if err != nil {

		if response.StatusCode == 404 {
			return nil, bmxerror.New(ErrCodeNoAccountExists,
				fmt.Sprintf("No account exists in the given region: %q and the given org: %q", region, orgGUID))
		}
		return nil, err

	}

	if len(queryResp.Accounts) > 0 {
		account := queryResp.Accounts[0].ToModel()
		return &account, nil
	}

	return nil, bmxerror.New(ErrCodeNoAccountExists,
		fmt.Sprintf("No account exists in the given region: %q and the given org: %q", region, orgGUID))
}

func (a *account) List() ([]Account, error) {
	var accounts []Account
	resp, err := a.client.GetPaginated("/coe/v2/accounts", NewAccountPaginatedResources(AccountResource{}), func(resource interface{}) bool {
		if accountResource, ok := resource.(AccountResource); ok {
			accounts = append(accounts, accountResource.ToModel())
			return true
		}
		return false
	})

	if resp.StatusCode == 404 || len(accounts) == 0 {
		return nil, bmxerror.New(ErrCodeNoAccountExists,
			fmt.Sprintf("No Account exists"))
	}

	return accounts, err
}

//FindByOwner ...
func (a *account) FindByOwner(userID string) (*Account, error) {
	accounts, err := a.List()
	if err != nil {
		return nil, err
	}

	for _, a := range accounts {
		if a.OwnerUserID == userID {
			return &a, nil
		}
	}
	return nil, bmxerror.New(ErrCodeNoAccountExists,
		fmt.Sprintf("No account exists for the user %q", userID))
}

//Get ...
func (a *account) Get(accountId string) (*Account, error) {
	queryResp := AccountNameQueryResponse{}
	response, err := a.client.Get(fmt.Sprintf("/coe/v2/accounts/%s", accountId), &queryResp)
	if err != nil {

		if response.StatusCode == 404 {
			return nil, bmxerror.New(ErrCodeNoAccountExists,
				fmt.Sprintf("Account %q does not exists", accountId))
		}
		return nil, err

	}

	account := queryResp.ToModel()
	return &account, nil

}
