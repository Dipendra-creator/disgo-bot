package shared

import "strings"

// idSep separates the segments of a component/modal custom ID.
const idSep = ":"

// BuildID encodes a custom ID of the form "<module>:<action>[:arg...]". The
// router uses the module segment to route the interaction and the action plus
// args are handed to the handler via Context.Args.
//
// NOTE: Discord limits custom IDs to 100 characters; keep args compact (IDs,
// short tokens) and store larger state in the cache/DB keyed by a short token.
func BuildID(module, action string, args ...string) string {
	parts := append([]string{module, action}, args...)
	return strings.Join(parts, idSep)
}

// ParseID decodes a custom ID produced by BuildID. The returned args are the
// segments following the action (nil when none).
func ParseID(id string) (module, action string, args []string) {
	parts := strings.Split(id, idSep)
	switch len(parts) {
	case 0:
		return "", "", nil
	case 1:
		return parts[0], "", nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return parts[0], parts[1], parts[2:]
	}
}
