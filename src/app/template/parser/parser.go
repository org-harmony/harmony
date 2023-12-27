package parser

import (
	"github.com/org-harmony/harmony/src/core/trans"
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
	TemplateID      string `json:"templateID,omitempty"`
	TemplateType    string
	TemplateVersion string `json:"templateVersion,omitempty"`
	TemplateName    string `json:"templateName,omitempty"`
	VariantName     string `json:"variantName,omitempty"`
	Requirement     string `json:"requirement,omitempty"`
	Errors          []ParsingLog
	Warnings        []ParsingLog
	Notices         []ParsingLog
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

// ViolationsForRule returns all violations (errors) for a given rule.
// This can be used to check if a rule was violated and what to display to the user.
// Warnings and Notices are not considered violations and will be displayed to the user as general information, not per rule.
func (r ParsingResult) ViolationsForRule(rule string) []ParsingLog {
	var violations []ParsingLog
	for _, log := range r.Errors {
		if log.Segment.Name != rule {
			continue
		}

		violations = append(violations, log)
	}

	return violations
}
