package structures

const (
	ProjectDefaultBranchName = "master"
)

type Project struct {
	Id               string
	Name             string
	Description      string
	RepositoryUrl    string
	RepositoryBranch string
	Inventory        string
	InventoryList    []string
	CollectionsList  []string
	Variables        string
	VariablesList    []string
	VariablesMain    bool
	VariablesVault   bool
	VaultPassword    string
}
