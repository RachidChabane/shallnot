package spec

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/RachidChabane/shallnot/internal/domain"
)

// Keys of the YAML spec format.
const (
	keyRequirements  = "requirements"
	keyID            = "id"
	keyRevision      = "revision"
	keyTitle         = "title"
	keyStatement     = "statement"
	keyStatus        = "status"
	keyJustification = "justification"
	keyOutOfScope    = "out_of_scope"
	keyDimension     = "dimension"
	keyReason        = "reason"
)

// YAMLParser reads the YAML spec format.
type YAMLParser struct {
	IDs domain.IDPattern
}

// Parse reads the requirements of one YAML document.
func (p YAMLParser) Parse(reader io.Reader, displayPath string) (Document, error) {
	var root yaml.Node
	if err := yaml.NewDecoder(reader).Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return Document{}, nil
		}
		return Document{}, fmt.Errorf("%s: %w", displayPath, err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return Document{}, fmt.Errorf("%s: a YAML spec is a mapping with a %q list", displayPath, keyRequirements)
	}
	var document Document
	problem := func(node *yaml.Node, format string, args ...any) {
		document.Problems = append(document.Problems, domain.SpecProblem{
			Location: domain.Location{File: displayPath, Line: node.Line},
			Message:  fmt.Sprintf(format, args...),
		})
	}
	mapping := root.Content[0]
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		key, value := mapping.Content[index], mapping.Content[index+1]
		switch key.Value {
		case keyRequirements:
			if value.Kind != yaml.SequenceNode {
				return Document{}, fmt.Errorf("%s:%d: %q must be a list", displayPath, value.Line, keyRequirements)
			}
			for _, item := range value.Content {
				if requirement, ok := p.parseItem(item, displayPath, problem); ok {
					document.Requirements = append(document.Requirements, requirement)
				}
			}
		case keyOutOfScope:
			document.OutOfScope = append(document.OutOfScope, parseOutOfScope(value, displayPath, problem)...)
		default:
			problem(key, "unknown key %q (a YAML spec has the keys %q and %q)", key.Value, keyRequirements, keyOutOfScope)
		}
	}
	return document, nil
}

type problemFunc func(node *yaml.Node, format string, args ...any)

// parseOutOfScope reads the list of out-of-scope declarations, reporting each unreadable item.
func parseOutOfScope(list *yaml.Node, displayPath string, problem problemFunc) []domain.OutOfScope {
	if list.Kind != yaml.SequenceNode {
		problem(list, "%q must be a list", keyOutOfScope)
		return nil
	}
	var declarations []domain.OutOfScope
	for _, item := range list.Content {
		if declaration, ok := parseOutOfScopeItem(item, displayPath, problem); ok {
			declarations = append(declarations, declaration)
		}
	}
	return declarations
}

func parseOutOfScopeItem(item *yaml.Node, displayPath string, problem problemFunc) (domain.OutOfScope, bool) {
	if item.Kind != yaml.MappingNode {
		problem(item, "an out-of-scope declaration must be a mapping")
		return domain.OutOfScope{}, false
	}
	declaration := domain.OutOfScope{Location: domain.Location{File: displayPath, Line: item.Line}}
	valid := true
	for index := 0; index+1 < len(item.Content); index += 2 {
		key, value := item.Content[index], item.Content[index+1]
		if value.Kind != yaml.ScalarNode {
			problem(value, "%q must be a scalar", key.Value)
			valid = false
			continue
		}
		text := strings.Join(strings.Fields(value.Value), " ")
		switch key.Value {
		case keyDimension:
			declaration.Dimension = text
		case keyReason:
			declaration.Reason = text
		default:
			problem(key, "unknown key %q in an out-of-scope declaration", key.Value)
			valid = false
		}
	}
	if message := outOfScopeProblem(declaration); valid && message != "" {
		problem(item, "%s", message)
		valid = false
	}
	return declaration, valid
}

func (p YAMLParser) parseItem(item *yaml.Node, displayPath string, problem problemFunc) (domain.Requirement, bool) {
	if item.Kind != yaml.MappingNode {
		problem(item, "a requirement must be a mapping")
		return domain.Requirement{}, false
	}
	requirement := domain.Requirement{
		Revision: domain.FirstRevision,
		Location: domain.Location{File: displayPath, Line: item.Line},
	}
	valid := true
	reject := func(node *yaml.Node, format string, args ...any) {
		problem(node, format, args...)
		valid = false
	}
	for index := 0; index+1 < len(item.Content); index += 2 {
		key, value := item.Content[index], item.Content[index+1]
		if value.Kind != yaml.ScalarNode {
			reject(value, "%q must be a scalar", key.Value)
			continue
		}
		text := strings.TrimSpace(value.Value)
		switch key.Value {
		case keyID:
			if !p.IDs.Matches(text) {
				reject(value, "id %q does not match the id pattern %s", text, p.IDs.Source())
			}
			requirement.ID = domain.RequirementID(text)
		case keyRevision:
			revision, err := domain.ParseRevision(text)
			if err != nil {
				reject(value, "%v", err)
			}
			requirement.Revision = revision
		case keyTitle:
			requirement.Title = text
		case keyStatement:
			requirement.Statement = strings.Join(strings.Fields(text), " ")
		case keyStatus:
			if err := requirement.Status.UnmarshalText([]byte(text)); err != nil {
				reject(value, "%v", err)
			}
		case keyJustification:
			requirement.Justification = strings.Join(strings.Fields(text), " ")
		default:
			reject(key, "unknown key %q", key.Value)
		}
	}
	if requirement.ID == "" && valid {
		reject(item, "a requirement needs an %q", keyID)
	}
	if requirement.Statement == "" && valid {
		reject(item, "requirement %s needs a %q", requirement.ID, keyStatement)
	}
	return requirement, valid
}
