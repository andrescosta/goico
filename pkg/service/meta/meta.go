package meta

import "time"

type Data struct {
	Name      string
	Version   string
	Kind      string
	BuildID   *string
	StartTime time.Time
	HnUpTime  string
	Env       map[string]string
}
