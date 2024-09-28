package data

import "context"

type SnuffleContext interface {
	// TODO: this is a rich wrapper around context.Context
	GetContext() context.Context
}
