// Package junitxml reads test results in the JUnit XML format emitted by
// pytest, jest-junit, Vitest, Surefire, Gradle and gotestsum.
package junitxml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const (
	rootSuites = "testsuites"
	rootSuite  = "testsuite"
)

type xmlSuites struct {
	Suites []xmlSuite `xml:"testsuite"`
}

type xmlSuite struct {
	Name   string     `xml:"name,attr"`
	Suites []xmlSuite `xml:"testsuite"`
	Cases  []xmlCase  `xml:"testcase"`
}

type xmlCase struct {
	Name       string        `xml:"name,attr"`
	ClassName  string        `xml:"classname,attr"`
	Failure    *struct{}     `xml:"failure"`
	Error      *struct{}     `xml:"error"`
	Skipped    *struct{}     `xml:"skipped"`
	Properties []xmlProperty `xml:"properties>property"`
}

type xmlProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	Text  string `xml:",chardata"`
}

// Reader turns JUnit XML files into domain test cases.
type Reader struct {
	Grammar domain.TagGrammar
}

// ReadFile reads one results file. displayPath is the path recorded on each test case.
func (r Reader) ReadFile(path, displayPath string) ([]domain.TestCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	testCases, err := r.Read(file, displayPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", displayPath, err)
	}
	return testCases, nil
}

// Read decodes a JUnit XML document.
func (r Reader) Read(reader io.Reader, displayPath string) ([]domain.TestCase, error) {
	decoder := xml.NewDecoder(reader)
	root, err := firstElement(decoder)
	if err != nil {
		return nil, err
	}
	var suites []xmlSuite
	switch root.Name.Local {
	case rootSuites:
		var document xmlSuites
		if err := decoder.DecodeElement(&document, &root); err != nil {
			return nil, fmt.Errorf("not valid JUnit XML: %w", err)
		}
		suites = document.Suites
	case rootSuite:
		var suite xmlSuite
		if err := decoder.DecodeElement(&suite, &root); err != nil {
			return nil, fmt.Errorf("not valid JUnit XML: %w", err)
		}
		suites = []xmlSuite{suite}
	default:
		return nil, fmt.Errorf("not JUnit XML: root element is <%s>, expected <%s> or <%s>", root.Name.Local, rootSuites, rootSuite)
	}
	var testCases []domain.TestCase
	for _, suite := range suites {
		testCases = r.collect(testCases, suite, displayPath)
	}
	return testCases, nil
}

func firstElement(decoder *xml.Decoder) (xml.StartElement, error) {
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return xml.StartElement{}, errors.New("not JUnit XML: the document is empty")
		}
		if err != nil {
			return xml.StartElement{}, fmt.Errorf("not valid XML: %w", err)
		}
		if start, isStart := token.(xml.StartElement); isStart {
			return start, nil
		}
	}
}

func (r Reader) collect(testCases []domain.TestCase, suite xmlSuite, displayPath string) []domain.TestCase {
	for _, testCase := range suite.Cases {
		testCases = append(testCases, r.convert(testCase, suite.Name, displayPath))
	}
	for _, nested := range suite.Suites {
		testCases = r.collect(testCases, nested, displayPath)
	}
	return testCases
}

func (r Reader) convert(testCase xmlCase, suiteName, displayPath string) domain.TestCase {
	converted := domain.TestCase{
		ResultsFile: displayPath,
		Suite:       suiteName,
		ClassName:   testCase.ClassName,
		Name:        testCase.Name,
		Outcome:     outcomeOf(testCase),
	}
	citations := newCitations()
	for _, text := range []string{testCase.Name, testCase.ClassName} {
		for _, match := range r.Grammar.FindBracketTags(text) {
			citations.add(match.Tag)
		}
	}
	for _, property := range testCase.Properties {
		if property.Name != domain.TagKeyword {
			continue
		}
		value := property.Value
		if value == "" {
			value = property.Text
		}
		citations.add(r.Grammar.ParseRefList(domain.TagKeyword+"="+value, value))
	}
	converted.Refs, converted.Malformed = citations.refs, citations.malformed
	return converted
}

func outcomeOf(testCase xmlCase) domain.Outcome {
	switch {
	case testCase.Error != nil:
		return domain.OutcomeErrored
	case testCase.Failure != nil:
		return domain.OutcomeFailed
	case testCase.Skipped != nil:
		return domain.OutcomeSkipped
	default:
		return domain.OutcomePassed
	}
}

// citations accumulates the distinct refs and malformed tags of one test case.
type citations struct {
	refs          []domain.Ref
	malformed     []domain.MalformedTag
	seenRefs      map[domain.Ref]bool
	seenMalformed map[domain.MalformedTag]bool
}

func newCitations() *citations {
	return &citations{seenRefs: map[domain.Ref]bool{}, seenMalformed: map[domain.MalformedTag]bool{}}
}

func (c *citations) add(tag domain.Tag) {
	for _, ref := range tag.Refs {
		if !c.seenRefs[ref] {
			c.seenRefs[ref] = true
			c.refs = append(c.refs, ref)
		}
	}
	for _, malformed := range tag.Malformed {
		if !c.seenMalformed[malformed] {
			c.seenMalformed[malformed] = true
			c.malformed = append(c.malformed, malformed)
		}
	}
}
