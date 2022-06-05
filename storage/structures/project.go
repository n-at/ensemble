package structures

import (
	"net/url"
	"strings"
)

const (
	ProjectDefaultBranchName    = "master"
	ProjectDefaultInventoryName = "main.yml"
)

type Project struct {
	Id                 string `db:"id"`
	Name               string `db:"name"`
	Description        string `db:"description"`
	RepositoryUrl      string `db:"repo_url"`
	RepositoryLogin    string `db:"repo_login"`
	RepositoryPassword string `db:"repo_password"`
	RepositoryBranch   string `db:"repo_branch"`
	Inventory          string `db:"inventory"`
	Inventories        string `db:"inventory_list"`
	Collections        string `db:"collections_list"`
	Variables          string `db:"variables"`
	VariablesAvailable string `db:"variables_list"`
	VariablesMain      bool   `db:"variables_main"`
	VariablesVault     bool   `db:"variables_vault"`
	VaultPassword      string `db:"vault_password"`
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

func (p *Project) InventoryList() []string {
	if len(p.Inventories) != 0 {
		return strings.Split(p.Inventories, "|")
	} else {
		return []string{}
	}
}

func (p *Project) CollectionsList() []string {
	if len(p.Collections) != 0 {
		return strings.Split(p.Collections, "|")
	} else {
		return []string{}
	}
}

func (p *Project) VariablesList() []string {
	if len(p.VariablesAvailable) != 0 {
		return strings.Split(p.VariablesAvailable, "|")
	} else {
		return []string{}
	}
}
