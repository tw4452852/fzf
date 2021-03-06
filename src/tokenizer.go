package fzf

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/junegunn/fzf/src/util"
)

const rangeEllipsis = 0

// Range represents nth-expression
type Range struct {
	begin int
	end   int
}

// Transformed holds the result of tokenization and transformation
type Transformed struct {
	whole *string
	parts []Token
}

// Token contains the tokenized part of the strings and its prefix length
type Token struct {
	text         *string
	prefixLength int
}

// ParseRange parses nth-expression and returns the corresponding Range object
func ParseRange(str *string) (Range, bool) {
	if (*str) == ".." {
		return Range{rangeEllipsis, rangeEllipsis}, true
	} else if strings.HasPrefix(*str, "..") {
		end, err := strconv.Atoi((*str)[2:])
		if err != nil || end == 0 {
			return Range{}, false
		}
		return Range{rangeEllipsis, end}, true
	} else if strings.HasSuffix(*str, "..") {
		begin, err := strconv.Atoi((*str)[:len(*str)-2])
		if err != nil || begin == 0 {
			return Range{}, false
		}
		return Range{begin, rangeEllipsis}, true
	} else if strings.Contains(*str, "..") {
		ns := strings.Split(*str, "..")
		if len(ns) != 2 {
			return Range{}, false
		}
		begin, err1 := strconv.Atoi(ns[0])
		end, err2 := strconv.Atoi(ns[1])
		if err1 != nil || err2 != nil {
			return Range{}, false
		}
		return Range{begin, end}, true
	}

	n, err := strconv.Atoi(*str)
	if err != nil || n == 0 {
		return Range{}, false
	}
	return Range{n, n}, true
}

func withPrefixLengths(tokens []string, begin int) []Token {
	ret := make([]Token, len(tokens))

	prefixLength := begin
	for idx, token := range tokens {
		// Need to define a new local variable instead of the reused token to take
		// the pointer to it
		str := token
		ret[idx] = Token{text: &str, prefixLength: prefixLength}
		prefixLength += len([]rune(token))
	}
	return ret
}

const (
	awkNil = iota
	awkBlack
	awkWhite
)

func awkTokenizer(input *string) ([]string, int) {
	// 9, 32
	ret := []string{}
	str := []rune{}
	prefixLength := 0
	state := awkNil
	for _, r := range []rune(*input) {
		white := r == 9 || r == 32
		switch state {
		case awkNil:
			if white {
				prefixLength++
			} else {
				state = awkBlack
				str = append(str, r)
			}
		case awkBlack:
			str = append(str, r)
			if white {
				state = awkWhite
			}
		case awkWhite:
			if white {
				str = append(str, r)
			} else {
				ret = append(ret, string(str))
				state = awkBlack
				str = []rune{r}
			}
		}
	}
	if len(str) > 0 {
		ret = append(ret, string(str))
	}
	return ret, prefixLength
}

// Tokenize tokenizes the given string with the delimiter
func Tokenize(str *string, delimiter *regexp.Regexp) []Token {
	if delimiter == nil {
		// AWK-style (\S+\s*)
		tokens, prefixLength := awkTokenizer(str)
		return withPrefixLengths(tokens, prefixLength)
	}
	tokens := delimiter.FindAllString(*str, -1)
	return withPrefixLengths(tokens, 0)
}

func joinTokens(tokens []Token) string {
	ret := ""
	for _, token := range tokens {
		ret += *token.text
	}
	return ret
}

// Transform is used to transform the input when --with-nth option is given
func Transform(tokens []Token, withNth []Range) *Transformed {
	transTokens := make([]Token, len(withNth))
	numTokens := len(tokens)
	whole := ""
	for idx, r := range withNth {
		part := ""
		minIdx := 0
		if r.begin == r.end {
			idx := r.begin
			if idx == rangeEllipsis {
				part += joinTokens(tokens)
			} else {
				if idx < 0 {
					idx += numTokens + 1
				}
				if idx >= 1 && idx <= numTokens {
					minIdx = idx - 1
					part += *tokens[idx-1].text
				}
			}
		} else {
			var begin, end int
			if r.begin == rangeEllipsis { // ..N
				begin, end = 1, r.end
				if end < 0 {
					end += numTokens + 1
				}
			} else if r.end == rangeEllipsis { // N..
				begin, end = r.begin, numTokens
				if begin < 0 {
					begin += numTokens + 1
				}
			} else {
				begin, end = r.begin, r.end
				if begin < 0 {
					begin += numTokens + 1
				}
				if end < 0 {
					end += numTokens + 1
				}
			}
			minIdx = util.Max(0, begin-1)
			for idx := begin; idx <= end; idx++ {
				if idx >= 1 && idx <= numTokens {
					part += *tokens[idx-1].text
				}
			}
		}
		whole += part
		var prefixLength int
		if minIdx < numTokens {
			prefixLength = tokens[minIdx].prefixLength
		} else {
			prefixLength = 0
		}
		transTokens[idx] = Token{&part, prefixLength}
	}
	return &Transformed{
		whole: &whole,
		parts: transTokens}
}
