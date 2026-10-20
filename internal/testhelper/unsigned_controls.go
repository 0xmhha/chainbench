package testhelper

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type unsignedControlRule struct {
	Minimum, Maximum uint64
	Default          *uint64
}

func unsignedRule(minimum, maximum uint64, defaults ...uint64) unsignedControlRule {
	rule := unsignedControlRule{Minimum: minimum, Maximum: maximum}
	if len(defaults) > 0 {
		value := defaults[0]
		rule.Default = &value
	}
	return rule
}

// unsignedControlRules owns the ranges of controls consumed as unsigned values.
// Runtime guards and generated editor schemas use the same definitions.
func unsignedControlRules(kind, name string) map[string]unsignedControlRule {
	maxInt := uint64(math.MaxInt)
	if kind == "action" {
		switch name {
		case actionWaitBlock:
			return map[string]unsignedControlRule{"target": unsignedRule(0, math.MaxUint64)}
		case actionSendTx:
			return map[string]unsignedControlRule{"blocks": unsignedRule(1, math.MaxUint64, defaultKeptOutBlocks)}
		case actionLoad:
			return map[string]unsignedControlRule{"blocks": unsignedRule(1, maxInt, uint64(loadDefaultBlocks)), "fillPercent": unsignedRule(1, 100, 100)}
		case actionReadNodeLog:
			return map[string]unsignedControlRule{"maxBytes": unsignedRule(1, maxInt, nodeLogDefaultMaxBytes)}
		}
	}
	if kind == "reader" || kind == "assertion" {
		switch name {
		case assertDerive, assertLogs:
			return map[string]unsignedControlRule{"index": unsignedRule(0, maxInt, 0)}
		case assertReceiptLog:
			return map[string]unsignedControlRule{"index": unsignedRule(0, maxInt, 0), "topic": unsignedRule(0, maxInt, 1)}
		case assertCreateAddress:
			return map[string]unsignedControlRule{"nonce": unsignedRule(0, math.MaxUint64)}
		}
	}
	if kind == "assertion" {
		switch name {
		case assertBlockLag:
			return map[string]unsignedControlRule{"maxLag": unsignedRule(0, math.MaxUint64, defaultBlockLagMax)}
		case assertBlockHalt:
			return map[string]unsignedControlRule{"maxAdvance": unsignedRule(0, math.MaxUint64, defaultBlockHaltMaxAdv)}
		case assertBlockInterval:
			return map[string]unsignedControlRule{
				"blocks":     unsignedRule(1, maxInt, defaultIntervalSamples),
				"maxSeconds": unsignedRule(0, 9007199254740991, defaultIntervalMaxSecs),
				"maxMillis":  unsignedRule(0, 9007199254740991, defaultIntervalMaxSecs*1000),
				"minSeconds": unsignedRule(0, 9007199254740991, defaultIntervalMinSecs),
				"minMillis":  unsignedRule(0, 9007199254740991, defaultIntervalMinSecs*1000),
			}
		case assertWSSubscribe, assertWSCollected:
			return map[string]unsignedControlRule{"count": unsignedRule(1, maxInt, 1)}
		}
	}
	return nil
}

func validateUnsignedControl(kind, name, field string, args map[string]any) error {
	value, present := args[field]
	if !present {
		return nil
	}
	rule, known := unsignedControlRules(kind, name)[field]
	if !known {
		return fmt.Errorf("dsl: unknown unsigned control %s.%s", name, field)
	}
	n, ok := uintArg(value)
	if !ok || n < rule.Minimum || n > rule.Maximum {
		return fmt.Errorf("dsl: bad %s: %s expects an exact integer from %d to %d", field, name, rule.Minimum, rule.Maximum)
	}
	return nil
}

func unsignedControlSchema(rule unsignedControlRule) map[string]any {
	maximum := min(rule.Maximum, uint64(9007199254740991))
	literal := boundedUnsignedPattern(rule.Minimum, rule.Maximum)
	schema := map[string]any{"anyOf": []any{
		map[string]any{"type": "integer", "minimum": int64(rule.Minimum), "maximum": int64(maximum)},
		map[string]any{"type": "string", "pattern": `^(?:\$[A-Za-z_][A-Za-z0-9_]*|.*\$\{[A-Za-z_][A-Za-z0-9_]*\}.*|` + literal + `)$`},
	}}
	if rule.Default != nil {
		schema["default"] = *rule.Default
	}
	return schema
}

// boundedUnsignedPattern accepts decimal/hex literals with optional leading
// zeros. Bounds are encoded in ordinary JSON Schema, without a second validator.
func boundedUnsignedPattern(minimum, maximum uint64) string {
	decimal := positiveUpToPattern(maximum, 10)
	hexadecimal := positiveUpToPattern(maximum, 16)
	if minimum == 0 {
		decimal = "(?:0|" + decimal + ")"
		hexadecimal = "(?:0|" + hexadecimal + ")"
	}
	return `(?:0*(?:` + decimal + `)|0[xX]0*(?:` + hexadecimal + `))`
}

func positiveUpToPattern(maximum uint64, base int) string {
	digits := strconv.FormatUint(maximum, base)
	alternatives := []string{}
	if len(digits) > 1 {
		alternatives = append(alternatives, digitPattern(1, base-1)+repeatedDigits(base, len(digits)-2, true))
	}
	prefix := ""
	for i, c := range digits {
		value, _ := strconv.ParseUint(string(c), base, 8)
		lower := 0
		if i == 0 {
			lower = 1
		}
		if int(value) > lower {
			alternatives = append(alternatives, prefix+digitPattern(lower, int(value)-1)+repeatedDigits(base, len(digits)-i-1, false))
		}
		prefix += digitPattern(int(value), int(value))
	}
	alternatives = append(alternatives, prefix)
	return strings.Join(alternatives, "|")
}

func repeatedDigits(base, count int, shorter bool) string {
	if count == 0 {
		return ""
	}
	repeat := "{" + strconv.Itoa(count) + "}"
	if shorter {
		repeat = "{0," + strconv.Itoa(count) + "}"
	}
	return digitPattern(0, base-1) + repeat
}

func digitPattern(from, to int) string {
	chars := ""
	for n := from; n <= to; n++ {
		digit := strconv.FormatInt(int64(n), 16)
		chars += digit
		if n >= 10 {
			chars += strings.ToUpper(digit)
		}
	}
	if len(chars) == 1 {
		return chars
	}
	return "[" + chars + "]"
}
