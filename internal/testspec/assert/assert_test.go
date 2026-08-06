package assert_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/testspec/assert"
)

func TestEqual_TypeAwareNumeric(t *testing.T) {
	// 7 (int), "7" (decimal), "0x7" (hex), 7.0 (json number) are all equal.
	forms := []any{7, "7", "0x7", 7.0}
	for i, a := range forms {
		for _, b := range forms {
			if pass, _ := assert.Equal(a, b); !pass {
				t.Fatalf("Equal(%v[%d], %v) should pass", a, i, b)
			}
		}
	}
	// Huge wei via hex string vs decimal string.
	if pass, _ := assert.Equal("0xde0b6b3a7640000", "1000000000000000000"); !pass {
		t.Fatal("wei hex vs decimal should be equal")
	}
	if pass, _ := assert.Equal("wbft", "wbft"); !pass {
		t.Fatal("string equal")
	}
	if pass, _ := assert.Equal(7, 8); pass {
		t.Fatal("7 != 8")
	}
}

func TestEqualCI_Address(t *testing.T) {
	a := "0xAbC0000000000000000000000000000000000001"
	b := "0xabc0000000000000000000000000000000000001"
	if pass, _ := assert.EqualCI(a, b); !pass {
		t.Fatal("addresses should be case-insensitively equal")
	}
	if pass, _ := assert.Equal(a, b); pass {
		t.Fatal("Equal should be case-sensitive for non-numeric strings")
	}
}

func TestLen(t *testing.T) {
	if pass, _ := assert.Len([]int{1, 2, 3, 4, 5, 6, 7}, 7); !pass {
		t.Fatal("len 7")
	}
	if pass, _ := assert.Len([]int{1, 2}, 7); pass {
		t.Fatal("len mismatch")
	}
	if pass, _ := assert.Len("abc", 3); !pass {
		t.Fatal("string len")
	}
}

func TestComparisons(t *testing.T) {
	if pass, _ := assert.Greater(10, 5); !pass {
		t.Fatal("10 > 5")
	}
	if pass, _ := assert.GreaterOrEqual(5, 5); !pass {
		t.Fatal("5 >= 5")
	}
	if pass, _ := assert.Less(3, 5); !pass {
		t.Fatal("3 < 5")
	}
	if pass, _ := assert.Greater(3, 5); pass {
		t.Fatal("3 not > 5")
	}
}

func TestInDelta(t *testing.T) {
	if pass, _ := assert.InDelta(100.0, 105.0, 10.0); !pass {
		t.Fatal("within delta")
	}
	if pass, _ := assert.InDelta(100.0, 120.0, 10.0); pass {
		t.Fatal("outside delta")
	}
}

func TestContainsRegexp(t *testing.T) {
	if pass, _ := assert.Contains("block reward paid", "reward"); !pass {
		t.Fatal("substring")
	}
	if pass, _ := assert.Contains([]any{"a", "b", "c"}, "b"); !pass {
		t.Fatal("slice element")
	}
	if pass, _ := assert.Regexp("block reward 100", `reward \d+`); !pass {
		t.Fatal("regexp match")
	}
}

func TestBoolNil(t *testing.T) {
	if pass, _ := assert.True(true, nil); !pass {
		t.Fatal("true")
	}
	if pass, _ := assert.False(false, nil); !pass {
		t.Fatal("false")
	}
	if pass, _ := assert.NotNil("x", nil); !pass {
		t.Fatal("not nil")
	}
	if pass, _ := assert.Nil(nil, nil); !pass {
		t.Fatal("nil")
	}
	var p *int
	if pass, _ := assert.Nil(p, nil); !pass {
		t.Fatal("typed nil pointer is nil")
	}
}

func TestElementsMatchAndHashes(t *testing.T) {
	if pass, _ := assert.ElementsMatch([]any{3, 1, 2}, []any{1, 2, 3}); !pass {
		t.Fatal("same multiset")
	}
	if pass, _ := assert.ElementsMatch([]any{1, 2}, []any{1, 2, 3}); pass {
		t.Fatal("different length")
	}
	if pass, _ := assert.HashesEqual([]string{"0xaa", "0xaa", "0xaa"}); !pass {
		t.Fatal("all equal hashes")
	}
	if pass, _ := assert.HashesEqual([]string{"0xaa", "0xbb"}); pass {
		t.Fatal("divergent hashes")
	}
}

func TestLookup(t *testing.T) {
	fn, ok := assert.Lookup("Len")
	if !ok {
		t.Fatal("Len must be registered")
	}
	if pass, _ := fn([]int{1, 2}, 2); !pass {
		t.Fatal("looked-up Len")
	}
	if _, ok := assert.Lookup("NoSuchAssert"); ok {
		t.Fatal("unknown assert must not resolve")
	}
}
