// Package baseline reads the statement digests of an earlier shallnot report.
package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const (
	supportedMajor = 1
	// firstMinorWithDigests is the first minor version of the report format that carries statement digests.
	firstMinorWithDigests = 2
	versionParts          = 2
)

type report struct {
	SchemaVersion *string `json:"schema_version"`
	Run           struct {
		StatementCanonicalization string `json:"statement_canonicalization"`
	} `json:"run"`
	Requirements *[]struct {
		ID              string `json:"id"`
		Revision        int    `json:"revision"`
		StatementDigest string `json:"statement_digest"`
	} `json:"requirements"`
}

// Read returns the statement digest of every requirement of the report file, by reference.
func Read(path string) (map[domain.Ref]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	digests, err := Parse(content)
	if err != nil {
		return nil, fmt.Errorf("baseline %s: %w", path, err)
	}
	return digests, nil
}

// Parse reads the statement digests of a JSON report. A report that predates
// digests, that lacks one or that computed them with another canonicalization
// is refused: comparing against it would compare nothing.
func Parse(content []byte) (map[domain.Ref]string, error) {
	var parsed report
	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, fmt.Errorf("not a JSON report: %w", err)
	}
	if parsed.SchemaVersion == nil || parsed.Requirements == nil {
		return nil, errors.New("not a shallnot report: it has no schema_version or no requirements")
	}
	if err := checkVersion(*parsed.SchemaVersion); err != nil {
		return nil, err
	}
	if canonicalization := parsed.Run.StatementCanonicalization; canonicalization != domain.StatementCanonicalization {
		return nil, fmt.Errorf("its statement digests use the canonicalization %q, not %q: they cannot be compared",
			canonicalization, domain.StatementCanonicalization)
	}
	digests := make(map[domain.Ref]string, len(*parsed.Requirements))
	for _, requirement := range *parsed.Requirements {
		if requirement.StatementDigest == "" {
			return nil, fmt.Errorf("requirement %s has no statement_digest", requirement.ID)
		}
		ref := domain.Ref{ID: domain.RequirementID(requirement.ID), Revision: domain.Revision(requirement.Revision)}
		digests[ref] = requirement.StatementDigest
	}
	return digests, nil
}

func checkVersion(version string) error {
	major, minor, found := strings.Cut(version, ".")
	majorNumber, majorErr := strconv.Atoi(major)
	minorNumber, minorErr := strconv.Atoi(minor)
	if !found || majorErr != nil || minorErr != nil || strings.Count(version, ".")+1 != versionParts {
		return fmt.Errorf("schema_version %q is not MAJOR.MINOR", version)
	}
	if majorNumber != supportedMajor || minorNumber < firstMinorWithDigests {
		return fmt.Errorf("schema_version %s carries no statement digests: the baseline must be a report of version %d.%d or later",
			version, supportedMajor, firstMinorWithDigests)
	}
	return nil
}
