package types

type VersionResp struct {
	BranchName string `json:"branchName"`
	BuildTime  string `json:"buildTime"`
	CommitHash string `json:"commitHash"`
}
