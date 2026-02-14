package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// diceRe matches expressions like 1d20, 2d6+3, 4d6dl1
var diceRe = regexp.MustCompile(`\[(\d+)d(\d+)(?:(dl|dh|kl|kh)(\d+))?([+-]\d+)?\]`)

type Result struct {
	Expression string
	Rolls      []int
	Dropped    []int
	Modifier   int
	Total      int
}

func (r Result) String() string {
	kept := make([]string, len(r.Rolls))
	for i, v := range r.Rolls {
		kept[i] = strconv.Itoa(v)
	}
	s := fmt.Sprintf("[%s = %s", r.Expression, strings.Join(kept, "+"))
	if len(r.Dropped) > 0 {
		dropped := make([]string, len(r.Dropped))
		for i, v := range r.Dropped {
			dropped[i] = strconv.Itoa(v)
		}
		s += " (dropped: " + strings.Join(dropped, ",") + ")"
	}
	if r.Modifier != 0 {
		if r.Modifier > 0 {
			s += fmt.Sprintf("+%d", r.Modifier)
		} else {
			s += fmt.Sprintf("%d", r.Modifier)
		}
	}
	s += fmt.Sprintf(" = %d]", r.Total)
	return s
}

// Process replaces all dice expressions in text with their results.
func Process(text string) string {
	return diceRe.ReplaceAllStringFunc(text, func(match string) string {
		r := Roll(match)
		if r == nil {
			return match
		}
		return r.String()
	})
}

// Roll parses and rolls a single dice expression like "[2d6+3]".
func Roll(expr string) *Result {
	m := diceRe.FindStringSubmatch(expr)
	if m == nil {
		return nil
	}

	count, _ := strconv.Atoi(m[1])
	sides, _ := strconv.Atoi(m[2])
	filterType := m[3]
	filterCount := 0
	if m[4] != "" {
		filterCount, _ = strconv.Atoi(m[4])
	}
	modifier := 0
	if m[5] != "" {
		modifier, _ = strconv.Atoi(m[5])
	}

	if count <= 0 || count > 100 || sides <= 0 || sides > 1000 {
		return nil
	}

	// Roll all dice
	all := make([]int, count)
	for i := range all {
		all[i] = rand.Intn(sides) + 1
	}

	// Sort for filtering
	sorted := make([]int, len(all))
	copy(sorted, all)
	sort.Ints(sorted)

	var kept, dropped []int

	switch filterType {
	case "dl": // drop lowest N
		if filterCount >= count {
			filterCount = count - 1
		}
		dropped = sorted[:filterCount]
		kept = sorted[filterCount:]
	case "dh": // drop highest N
		if filterCount >= count {
			filterCount = count - 1
		}
		kept = sorted[:count-filterCount]
		dropped = sorted[count-filterCount:]
	case "kl": // keep lowest N
		if filterCount > count {
			filterCount = count
		}
		kept = sorted[:filterCount]
		dropped = sorted[filterCount:]
	case "kh": // keep highest N
		if filterCount > count {
			filterCount = count
		}
		kept = sorted[count-filterCount:]
		dropped = sorted[:count-filterCount]
	default:
		kept = all
	}

	total := modifier
	for _, v := range kept {
		total += v
	}

	// Clean expression (remove brackets)
	cleanExpr := expr
	if len(cleanExpr) > 0 && cleanExpr[0] == '[' {
		cleanExpr = cleanExpr[1:]
	}
	if len(cleanExpr) > 0 && cleanExpr[len(cleanExpr)-1] == ']' {
		cleanExpr = cleanExpr[:len(cleanExpr)-1]
	}

	return &Result{
		Expression: cleanExpr,
		Rolls:      kept,
		Dropped:    dropped,
		Modifier:   modifier,
		Total:      total,
	}
}
