package docker2aci

type RepoData struct {
	Tokens    []string
	Endpoints []string
	Cookie    []string
}

type ParsedDockerURL struct {
	IndexURL  string
	ImageName string
	Tag       string
}
