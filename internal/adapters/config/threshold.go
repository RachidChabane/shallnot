package config

import (
	"fmt"
	"sort"

	"go.yaml.in/yaml/v3"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// thresholdDefaultKey is the key that holds the threshold of every confidence source the mapping does not name.
const thresholdDefaultKey = "default"

// ThresholdFile is a threshold of the `proofs` block: a bare number, which is the default for every confidence source,
// or a mapping from `default` and from confidence sources to numbers.
type ThresholdFile struct {
	Default  *float64
	BySource map[string]float64
}

// UnmarshalYAML reads a number or a mapping of numbers.
func (t *ThresholdFile) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var value float64
		if err := node.Decode(&value); err != nil {
			return fmt.Errorf("line %d: a threshold is a number or a mapping: %w", node.Line, err)
		}
		t.Default = &value
		return nil
	case yaml.MappingNode:
		entries := map[string]float64{}
		if err := node.Decode(&entries); err != nil {
			return fmt.Errorf("line %d: a threshold mapping holds numbers: %w", node.Line, err)
		}
		if value, set := entries[thresholdDefaultKey]; set {
			t.Default = &value
			delete(entries, thresholdDefaultKey)
		}
		t.BySource = entries
		return nil
	}
	return fmt.Errorf("line %d: a threshold is a number or a mapping", node.Line)
}

// apply resolves the threshold over the one in target, rejecting a value outside [0,1] and a key that names no
// confidence source.
func (t *ThresholdFile) apply(name string, target *proof.Threshold) error {
	if t == nil {
		return nil
	}
	if t.Default != nil {
		if err := checkConfidence(name, *t.Default); err != nil {
			return err
		}
		target.Default = *t.Default
	}
	keys := make([]string, 0, len(t.BySource))
	for key := range t.BySource {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		source, err := proof.ParseThresholdSource(key)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := checkConfidence(name+"."+key, t.BySource[key]); err != nil {
			return err
		}
		if target.BySource == nil {
			target.BySource = map[proof.ConfidenceSource]float64{}
		}
		target.BySource[source] = t.BySource[key]
	}
	return nil
}

func checkConfidence(name string, value float64) error {
	if value < proof.MinConfidence || value > proof.MaxConfidence {
		return fmt.Errorf("%s must lie in [%v,%v], got %v", name, proof.MinConfidence, proof.MaxConfidence, value)
	}
	return nil
}
