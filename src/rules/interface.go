package rules

import "snuffle/src/data"

type RulesEngine interface {
	Evaluate(ctx data.SnuffleContext, event *data.SnuffleEvent) (data.Rule, error)
	Run(ctx data.SnuffleContext, event *data.SnuffleEvent) (data.Rule, error)
}
