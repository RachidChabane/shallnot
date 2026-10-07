package config

import (
	"fmt"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// config resolves the block against the defaults, rejecting a value outside its closed set or range.
func (f ProofsFile) config() (proof.Config, error) {
	config := proof.DefaultConfig()
	if f.VerifiedDir != "" {
		config.VerifiedDir = f.VerifiedDir
	}
	for _, entry := range f.Require {
		kind, err := proof.ParseRequired(entry)
		if err != nil {
			return config, err
		}
		config.Require = appendOnce(config.Require, kind)
	}
	config.Code = f.Code
	if err := f.applyClosedSets(&config); err != nil {
		return config, err
	}
	if f.Thresholds != nil {
		if err := f.Thresholds.Validity.apply("thresholds.validity", &config.Thresholds.Validity); err != nil {
			return config, err
		}
		if err := f.Thresholds.Alignment.apply("thresholds.alignment", &config.Thresholds.Alignment); err != nil {
			return config, err
		}
	}
	if f.MaxHostExclusions != nil {
		if *f.MaxHostExclusions < 0 {
			return config, fmt.Errorf("max_host_exclusions must not be negative, got %d", *f.MaxHostExclusions)
		}
		config.MaxHostExclusions = *f.MaxHostExclusions
	}
	return config, nil
}

func (f ProofsFile) applyClosedSets(config *proof.Config) (err error) {
	if f.Trust != "" {
		if config.Trust, err = proof.ParseTrust(f.Trust); err != nil {
			return err
		}
	}
	if f.Escalation != "" {
		if config.Escalation, err = proof.ParseEscalation(f.Escalation); err != nil {
			return err
		}
	}
	if f.SignedPassBasis != "" {
		if config.SignedPassBasis, err = proof.ParseSignedPassBasis(f.SignedPassBasis); err != nil {
			return err
		}
	}
	if f.MinIsolation != "" {
		if config.MinIsolation, err = proof.ParseIsolation(f.MinIsolation); err != nil {
			return err
		}
	}
	return nil
}

func appendOnce(kinds []proof.Kind, kind proof.Kind) []proof.Kind {
	for _, existing := range kinds {
		if existing == kind {
			return kinds
		}
	}
	return append(kinds, kind)
}
