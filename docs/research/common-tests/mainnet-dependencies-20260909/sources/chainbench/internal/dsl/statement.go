package dsl

// ActionName returns the single key of an action entry. Action entries are
// {name: args} by schema; an empty entry yields "". Both the grammar (v1
// desugaring) and the interpreter read an entry's name through this one rule.
func ActionName(entry map[string]any) string {
	for k := range entry {
		return k
	}
	return ""
}

// ArgsOf returns the arguments map for an action value, or an empty map when the
// value is not a map (e.g. a bare boolean like {"ensureChain": true}).
func ArgsOf(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// SequenceOf returns the spec's unified statement sequence, deriving it from
// the legacy steps/assertions fields when the parser did not populate it
// (tests construct Spec directly). It is pure grammar — a v1 spec desugared
// into the same Statement list a v2 spec declares.
func SequenceOf(s Spec) []Statement {
	if len(s.Sequence) > 0 {
		return s.Sequence
	}
	out := make([]Statement, 0, len(s.Steps)+len(s.Assertions))
	for _, st := range s.Steps {
		out = append(out, Statement{Do: ActionName(st), Args: ArgsOf(st[ActionName(st)])})
	}
	for _, as := range s.Assertions {
		name, _ := as["assert"].(string)
		args := make(map[string]any, len(as))
		for k, v := range as {
			if k != "assert" {
				args[k] = v
			}
		}
		out = append(out, Statement{Expect: name, Args: args})
	}
	return out
}
