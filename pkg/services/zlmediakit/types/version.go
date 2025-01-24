package types

type VersionResp = Data[VersionRespRaw]

type VersionRespRaw struct {
	BranchName string `json:"branchName"`
	BuildTime  string `json:"buildTime"`
	CommitHash string `json:"commitHash"`
}
