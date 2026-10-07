package proof

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf16"
)

// SHA256Hex returns the lowercase hex SHA-256 digest of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// IsSHA256Hex reports whether s is a lowercase hex-encoded SHA-256 digest.
func IsSHA256Hex(s string) bool {
	if len(s) != Sha256HexLength {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// canonicalJSON writes value in the RFC 8785 canonical form. Only the JSON
// types of the cache key are supported: objects, arrays, strings and booleans;
// the cache key holds no number.
func canonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	if err := writeCanonical(&buffer, value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeCanonical(buffer *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		buffer.WriteString("null")
	case bool:
		fmt.Fprintf(buffer, "%t", typed)
	case string:
		writeCanonicalString(buffer, typed)
	case []any:
		buffer.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				buffer.WriteByte(',')
			}
			if err := writeCanonical(buffer, item); err != nil {
				return err
			}
		}
		buffer.WriteByte(']')
	case map[string]any:
		return writeCanonicalObject(buffer, typed)
	default:
		return fmt.Errorf("canonical JSON: unsupported value of type %T", value)
	}
	return nil
}

func writeCanonicalObject(buffer *bytes.Buffer, object map[string]any) error {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, compareUTF16)
	buffer.WriteByte('{')
	for index, key := range keys {
		if index > 0 {
			buffer.WriteByte(',')
		}
		writeCanonicalString(buffer, key)
		buffer.WriteByte(':')
		if err := writeCanonical(buffer, object[key]); err != nil {
			return err
		}
	}
	buffer.WriteByte('}')
	return nil
}

// compareUTF16 orders strings by UTF-16 code units, the key order RFC 8785 requires.
func compareUTF16(a, b string) int {
	left, right := utf16.Encode([]rune(a)), utf16.Encode([]rune(b))
	return slices.Compare(left, right)
}

const (
	firstPrintable = 0x20
	hexDigits      = "0123456789abcdef"
)

// writeCanonicalString escapes a string as ECMAScript's JSON.stringify does.
func writeCanonicalString(buffer *bytes.Buffer, text string) {
	buffer.WriteByte('"')
	for _, r := range text {
		switch {
		case r == '"':
			buffer.WriteString(`\"`)
		case r == '\\':
			buffer.WriteString(`\\`)
		case r == '\b':
			buffer.WriteString(`\b`)
		case r == '\f':
			buffer.WriteString(`\f`)
		case r == '\n':
			buffer.WriteString(`\n`)
		case r == '\r':
			buffer.WriteString(`\r`)
		case r == '\t':
			buffer.WriteString(`\t`)
		case r < firstPrintable:
			buffer.WriteString(`\u00`)
			buffer.WriteByte(hexDigits[r>>4])
			buffer.WriteByte(hexDigits[r&0xf])
		default:
			buffer.WriteRune(r)
		}
	}
	buffer.WriteByte('"')
}

// CacheKey returns the SHA-256 over the canonical JSON of the subject and the
// inputs. Both arrays are sorted first, so the key does not depend on their order.
func CacheKey(subject []Subject, inputs []Input) (string, error) {
	sortedInputs := slices.Clone(inputs)
	slices.SortStableFunc(sortedInputs, compareInputs)
	sortedSubject := slices.Clone(subject)
	slices.SortStableFunc(sortedSubject, compareSubjects)
	raw, err := json.Marshal(struct {
		Inputs  []Input   `json:"inputs"`
		Subject []Subject `json:"subject"`
	}{Inputs: sortedInputs, Subject: sortedSubject})
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return "", fmt.Errorf("cache key: %w", err)
	}
	canonical, err := canonicalJSON(generic)
	if err != nil {
		return "", err
	}
	return SHA256Hex(canonical), nil
}

func compareInputs(a, b Input) int {
	if byRole := strings.Compare(string(a.Role), string(b.Role)); byRole != 0 {
		return byRole
	}
	if byName := strings.Compare(a.Name, b.Name); byName != 0 {
		return byName
	}
	return strings.Compare(a.Digest.SHA256, b.Digest.SHA256)
}

func compareSubjects(a, b Subject) int {
	if byName := strings.Compare(a.Name, b.Name); byName != 0 {
		return byName
	}
	return strings.Compare(a.Digest.SHA256, b.Digest.SHA256)
}
