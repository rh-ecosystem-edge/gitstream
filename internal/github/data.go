package github

type Commit struct {
	SHA string
}

type ProcessError struct {
	Output     string
	ReturnCode *int
}

type BaseData struct {
	AppName     string
	Commit      Commit
	UpstreamURL string
}

type IssueData struct {
	BaseData
	Error ProcessError
}

type PRData BaseData
