// Package eiffel contains necessary functionality for the Elicitation Interface for eFFective Language (EIFFEL).
package eiffel

// Cfg is EIFFEL's configuration struct. This can be used to unmarshal a TOML configuration file into.
type Cfg struct {
	NeglectOptional bool `toml:"neglect_optional"`
}

// TODO add tests for service, web and output
