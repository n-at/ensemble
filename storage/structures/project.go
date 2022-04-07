package structures

type Project struct {
	id               string
	name             string
	description      string
	repositoryUrl    string
	repositoryBranch string
	inventory        string
	inventoryList    []string
	variables        string
	variablesList    []string
	vaultPassword    string
}
