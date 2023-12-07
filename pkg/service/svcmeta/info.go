package svcmeta

type Info struct {
	Name    string
	Version string
	Type    string
	BuildId *string
	Env     map[string]string
}
