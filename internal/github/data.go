package github

type Commit struct {
	SHA string
}

type ProcessError struct {
	Output     string
	ReturnCode *int
}

type Repository struct {
	Name RepoName
	URL  string
}

type IssueData struct {
	AppName  string
	Commit   Commit
	Error    ProcessError
	Upstream Repository
}
