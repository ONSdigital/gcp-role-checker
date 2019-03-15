package checker

import "encoding/json"

// StrSet ...
type StrSet struct {
	set map[string]bool
}

// Add ...
func (s *StrSet) Add(items ...string) {
	if s.set == nil {
		s.set = map[string]bool{}
	}
	for _, i := range items {
		s.set[i] = true
	}
}

// Contains ...
func (s *StrSet) Contains(item string) bool {
	_, ok := s.set[item]
	return ok
}

// AsSlice ...
func (s *StrSet) AsSlice() []string {
	keys := make([]string, len(s.set))

	i := 0
	for k := range s.set {
		keys[i] = k
		i++
	}

	return keys
}

// MarshalJSON ...
func (s StrSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.AsSlice())
}
