package parser

import (
	"context"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/validation"
)

const (
	// ParsingLogLevelError is the error level of a parsing log.
	ParsingLogLevelError ParsingLogLevel = iota
	// ParsingLogLevelWarning is the warning level of a parsing log.
	ParsingLogLevelWarning
	// ParsingLogLevelNotice is the notice level of a parsing log.
	ParsingLogLevelNotice
)

// ParsingLogLevel is the level of a parsing log.
type ParsingLogLevel int

// ParsingSegment is a part of an input string. The input string represents a requirement that is to be parsed.
// Each segment can be viewed as a token, usually put in via an input field.
// All segments together will be passed as a slice to the parser and parsed using the template.
type ParsingSegment struct {
	Name  string
	Value string
}

// ParsingResult is the result of parsing a requirement using a template.
type ParsingResult struct {
	TemplateID   string
	TemplateType string
	TemplateName string
	VariantName  string
	Errors       []ParsingLog
	Warnings     []ParsingLog
	Notices      []ParsingLog
}

// ParsingLog is a log entry of a parsing result. It contains the segment that was parsed, the level of the log and a message.
// It is used in the ParsingResult to report information about the parsing process.
type ParsingLog struct {
	Segment         *ParsingSegment
	Level           ParsingLogLevel
	Message         string
	TranslationArgs []string
	// Extra contains additional information about the log.
	// This field might be filled by custom parsers or rules.
	Extra map[string]any
	// Downgrade indicates that the parsing log was downgraded to a lower level.
	// This is usually the case when parsing errors occur on optional rules.
	Downgrade bool
}

// ParsableTemplate is a template that can be validated and parsed.
type ParsableTemplate interface {
	Validate(v validation.V) []error
	Parse(ctx context.Context, variation string, segments ...ParsingSegment) (ParsingResult, error)
}

// String on ParsingLog returns the message of the log.
func (l ParsingLog) String() string {
	return l.Message
}

// Translate on ParsingLog translates the message of the log.
func (l ParsingLog) Translate(translator trans.Translator) string {
	return translator.Tf(l.Message, l.TranslationArgs...)
}

// Ok returns true if the parsing result has no errors.
func (r ParsingResult) Ok() bool {
	return len(r.Errors) == 0
}

// Flawless returns true if the parsing result has no errors and no warnings.
func (r ParsingResult) Flawless() bool {
	return len(r.Errors) == 0 && len(r.Warnings) == 0
}
