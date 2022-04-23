package structures

import "net/url"

const (
	ProjectDefaultBranchName = "master"
)

type Project struct {
	Id                 string
	Name               string
	Description        string
	RepositoryUrl      string
	RepositoryLogin    string
	RepositoryPassword string
	RepositoryBranch   string
	Inventory          string
	InventoryList      []string
	CollectionsList    []string
	Variables          string
	VariablesList      []string
	VariablesMain      bool
	VariablesVault     bool
	VaultPassword      string
}

func (p *Project) RepositoryUrlFull() string {
	u, err := url.Parse(p.RepositoryUrl)
	if err != nil {
		return p.RepositoryUrl
	}

	if len(p.RepositoryLogin) != 0 {
		u.User = url.UserPassword(p.RepositoryLogin, p.RepositoryPassword)
	}

	return u.String()
}
