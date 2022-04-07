package structures

type Project struct {
	Id               string
	Name             string
	Description      string
	RepositoryUrl    string
	RepositoryBranch string
	Inventory        string
	InventoryList    []string
	Variables        string
	VariablesList    []string
	VaultPassword    string
}
