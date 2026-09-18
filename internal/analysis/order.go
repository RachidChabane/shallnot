package analysis

import (
	"sort"

	"github.com/RachidChabane/shallnot/internal/domain"
)

func locationLess(left, right domain.Location) bool {
	if left.File != right.File {
		return left.File < right.File
	}
	return left.Line < right.Line
}

func sortBoundTests(tests []domain.BoundTest) {
	sort.SliceStable(tests, func(i, j int) bool {
		left, right := tests[i], tests[j]
		switch {
		case left.ResultsFile != right.ResultsFile:
			return left.ResultsFile < right.ResultsFile
		case left.ClassName != right.ClassName:
			return left.ClassName < right.ClassName
		case left.Name != right.Name:
			return left.Name < right.Name
		default:
			return locationLess(locationOfTest(left), locationOfTest(right))
		}
	})
}

func sortFindings(findings []domain.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		switch {
		case left.Location != right.Location:
			return locationLess(left.Location, right.Location)
		case left.Category != right.Category:
			return left.Category < right.Category
		default:
			return left.Message < right.Message
		}
	})
}
