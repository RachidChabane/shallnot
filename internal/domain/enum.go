package domain

import "fmt"

// enumText returns the wire name of an enum value, given the ordered names of its type.
func enumText[T ~uint8](kind string, names []string, value T) ([]byte, error) {
	if int(value) >= len(names) {
		return nil, fmt.Errorf("invalid %s value %d", kind, value)
	}
	return []byte(names[value]), nil
}

// parseEnum resolves a wire name to its enum value.
func parseEnum[T ~uint8](kind string, names []string, text string) (T, error) {
	for index, name := range names {
		if name == text {
			return T(index), nil
		}
	}
	return 0, fmt.Errorf("unknown %s %q (expected one of %v)", kind, text, names)
}

func enumString[T ~uint8](kind string, names []string, value T) string {
	text, err := enumText(kind, names, value)
	if err != nil {
		return fmt.Sprintf("%s(%d)", kind, value)
	}
	return string(text)
}
