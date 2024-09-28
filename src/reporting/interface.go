package reporting

import "snuffle/src/data"

type Reporter interface {
	Report(event *data.SnuffleEvent, matched *data.Rule) error
}
