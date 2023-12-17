package meta

type Data struct {
	Name    string
	Version string
	Kind    string
	BuildID *string
	Env     map[string]string
}
