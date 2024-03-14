package table

import "strings"

// SanitizeID ensures the ID is not empty and consists of only lowercase alphanumeric characters. If permitLeadingDigits
// is false, then leading digits are stripped. A list of reserved values can be passed in to disallow specific IDs.
func SanitizeID(id string, permitLeadingDigits bool, reserved ...string) string {
	var buffer strings.Builder
	buffer.Grow(len(id))
	for _, ch := range id {
		if ch >= 'A' && ch <= 'Z' {
			ch += 'a' - 'A'
		}
		if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9' && (permitLeadingDigits || buffer.Len() > 0)) {
			buffer.WriteRune(ch)
		}
	}
	if buffer.Len() == 0 {
		buffer.WriteByte('_')
	}
	for {
		ok := true
		id = buffer.String()
		for _, one := range reserved {
			if one == id {
				buffer.WriteByte('_')
				ok = false
				break
			}
		}
		if ok {
			return id
		}
	}
}
