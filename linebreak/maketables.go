// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Unicode table generator. Based on unicode/maketables.go.
// Data read from the web.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func main() {
	flag.Parse()
	printHeader()
	printClasses()
	printTables()
	printSizes()
}

var (
	dataURL = flag.String("data", "",
		"full URL for LineBreak.txt; defaults to --url/LineBreak.txt")
	url = flag.String("url",
		"http://www.unicode.org/Public/6.2.0/ucd/",
		"URL of Unicode database directory")
	excludeclasses = flag.String("excludeclasses",
		"XX",
		"comma-separated list of (uppercase, two-letter) line breaking classes to ignore; default to XX")
	localFiles = flag.Bool("local", false,
		"data files have been copied to current directory; for debugging only")
)

var logger = log.New(os.Stderr, "", log.Lshortfile)

type class struct {
	name, doc string
}

// Supported line breaking classes for Unicode 6.2.0
//
// Table loading depends on this: classes not listed here aren't loaded.
var classes = []class{
	{"OP", "Open Punctuation"},
	{"CL", "Close Punctuation"},
	{"CP", "Close Parenthesis"},
	{"QU", "Quotation"},
	{"GL", "Non-breaking (\"Glue\")"},
	{"NS", "Nonstarter"},
	{"EX", "Exclamation/Interrogation"},
	{"SY", "Symbols Allowing Break After"},
	{"IS", "Infix Numeric Separator"},
	{"PR", "Prefix Numeric"},
	{"PO", "Postfix Numeric"},
	{"NU", "Numeric"},
	{"AL", "Alphabetic"},
	{"HL", "Hebrew Letter"},
	{"ID", "Ideographic"},
	{"IN", "Inseparable"},
	{"HY", "Hyphen"},
	{"BA", "Break After"},
	{"BB", "Break Before"},
	{"B2", "Break Opportunity Before and After"},
	{"ZW", "Zero Width Space"},
	{"CM", "Combining Mark"},
	{"WJ", "Word Joiner"},
	{"H2", "Hangul LV Syllable"},
	{"H3", "Hangul LVT Syllable"},
	{"JL", "Hangul L Jamo"},
	{"JV", "Hangul V Jamo"},
	{"JT", "Hangul T Jamo"},
	{"RI", "Regional Indicator"},
	// Resolved outside of the pair table (> 28).
	{"BK", "Mandatory Break"},
	{"CR", "Carriage Return"},
	{"LF", "Line Feed"},
	{"NL", "Next Line"},
	{"SG", "Surrogate"},
	{"SP", "Space"},
	{"CB", "Contingent Break Opportunity"},
	{"AI", "Ambiguous (Alphabetic or Ideographic)"},
	{"CJ", "Conditional Japanese Starter"},
	{"SA", "Complex Context Dependent (South East Asian)"},
	{"XX", "Unknown"},
}

var pairTableSize = 29

func allClassNames() []string {
	a := make([]string, 0, len(classes))
	for _, c := range classes {
		a = append(a, c.name)
	}
	sort.Strings(a)
	return a
}

type reader struct {
	*bufio.Reader
	fd   *os.File
	resp *http.Response
}

func open(url string) *reader {
	file := filepath.Base(url)
	if *localFiles {
		fd, err := os.Open(file)
		if err != nil {
			logger.Fatal(err)
		}
		return &reader{bufio.NewReader(fd), fd, nil}
	}
	resp, err := http.Get(url)
	if err != nil {
		logger.Fatal(err)
	}
	if resp.StatusCode != 200 {
		logger.Fatalf("bad GET status for %s: %d", file, resp.Status)
	}
	return &reader{bufio.NewReader(resp.Body), nil, resp}
}

func (r *reader) close() {
	if r.fd != nil {
		r.fd.Close()
	} else {
		r.resp.Body.Close()
	}
}

// codePoint represents a code point (or range of code points) for a line
// breaking class.
type codePoint struct {
	lo, hi uint32 // range of code points
	class  string
}

// Extract the version number from the URL
func version() string {
	// Break on slashes and look for the first numeric field
	fields := strings.Split(*url, "/")
	for _, f := range fields {
		if len(f) > 0 && '0' <= f[0] && f[0] <= '9' {
			return f
		}
	}
	logger.Fatal("unknown version")
	return "Unknown"
}

var codePointRe = regexp.MustCompile(`^([0-9A-F]+)(\.\.[0-9A-F]+)?;([A-Z0-9]{2})$`)

// LineBreak.txt has form:
//  4DFF;AL # HEXAGRAM FOR BEFORE COMPLETION
//  4E00..9FCC;ID # <CJK Ideograph, First>..<CJK Ideograph, Last>
func parseCodePoint(line string, codePoints map[string][]codePoint) {
	comment := strings.Index(line, "#")
	if comment >= 0 {
		line = line[0:comment]
	}
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	field := strings.Split(line, ";")
	if len(field) != 2 {
		logger.Fatalf("%s: %d fields (expected 2)\n", line, len(field))
	}
	matches := codePointRe.FindStringSubmatch(line)
	if len(matches) != 4 {
		logger.Fatalf("%s: %d matches (expected 3)\n", line, len(matches))
	}
	lo, err := strconv.ParseUint(matches[1], 16, 64)
	if err != nil {
		logger.Fatalf("%.5s...: %s", line, err)
	}
	hi := lo
	if len(matches[2]) > 2 { // ignore leading ..
		hi, err = strconv.ParseUint(matches[2][2:], 16, 64)
		if err != nil {
			logger.Fatalf("%.5s...: %s", line, err)
		}
	}
	name := matches[3]
	codePoints[name] = append(codePoints[name], codePoint{uint32(lo), uint32(hi), name})
}

func loadCodePoints(codePoints map[string][]codePoint) {
	if *dataURL == "" {
		flag.Set("data", *url+"LineBreak.txt")
	}
	input := open(*dataURL)
	for {
		line, err := input.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			logger.Fatal(err)
		}
		parseCodePoint(line[0:len(line)-1], codePoints)
	}
	input.close()
}

func excludeClass(class string, excludelist []string) bool {
	for _, name := range excludelist {
		if name == class {
			return true
		}
	}
	return false
}

const header = `// Generated by maketables.go
// DO NOT EDIT

package linebreak

import (
	"unicode"
)

// Version is the Unicode edition from which the tables are derived.
const Version = %q`

func printHeader() {
	fmt.Printf(header+"\n\n", version())
}

func printClasses() {
	fmt.Print("type breakClass int\n\n")
	fmt.Print("// Line breaking classes.\n")
	fmt.Print("//\n")
	fmt.Print("// See: http://www.unicode.org/reports/tr14/#Table1\n")
	fmt.Print("const (\n")
	for k, v := range classes {
		if k == 0 {
			fmt.Printf("\tClass%s breakClass = iota // %s\n", v.name, v.doc)
		} else {
			fmt.Printf("\tClass%s                   // %s\n", v.name, v.doc)
		}
		if k+1 == pairTableSize && k < len(classes)-1 {
			fmt.Printf("\t// Resolved outside of the pair table (> %d).\n", k)
		}
	}
	fmt.Print(")\n\n")

	fmt.Print("// Class returns the line breaking class for the given rune.\n")
	fmt.Print("func Class(r rune) breakClass {\n")
	fmt.Print("\t// TODO test more common first?\n")
	fmt.Print("\tswitch {\n")

	excludelist := strings.Split(strings.ToUpper(*excludeclasses), ",")
	for _, name := range allClassNames() {
		if excludeClass(name, excludelist) {
			continue
		}
		fmt.Printf("\tcase unicode.Is(%s, r):\n", name)
		fmt.Printf("\t\treturn Class%s\n", name)
	}
	fmt.Print("\t}\n")
	fmt.Print("\treturn ClassXX\n")
	fmt.Print("}\n\n")
}

func printTables() {
	excludelist := strings.Split(strings.ToUpper(*excludeclasses), ",")

	var list []string
	for _, name := range allClassNames() {
		if !excludeClass(name, excludelist) {
			list = append(list, name)
		}
	}

	codePoints := make(map[string][]codePoint)
	loadCodePoints(codePoints)

	decl := make(sort.StringSlice, len(list))
	ndecl := 0
	for _, name := range list {
		cp, ok := codePoints[name]
		if !ok {
			continue
		}
		decl[ndecl] = fmt.Sprintf(
			"\t%s = _%s; // %s is the set of Unicode characters in line breaking class %s.\n",
			name, name, name, name)
		ndecl++
		fmt.Printf("var _%s = &RangeTable {\n", name)
		ranges := foldAdjacent(cp)
		fmt.Print("\tR16: []Range16{\n")
		size := 16
		count := &range16Count
		for _, s := range ranges {
			size, count = printRange(s.Lo, s.Hi, s.Stride, size, count)
		}
		fmt.Print("\t},\n")
		if off := findLatinOffset(ranges); off > 0 {
			fmt.Printf("\tLatinOffset: %d,\n", off)
		}
		fmt.Print("}\n\n")
	}
	decl.Sort()
	fmt.Println("// These variables have type *RangeTable.")
	fmt.Println("var (")
	for _, d := range decl {
		fmt.Print(d)
	}
	fmt.Print(")\n\n")
}

// Tables may have a lot of adjacent elements. Fold them together.
func foldAdjacent(r []codePoint) []unicode.Range32 {
	s := make([]unicode.Range32, 0, len(r))
	j := 0
	for i := 0; i < len(r); i++ {
		if j > 0 && r[i].lo == s[j-1].Hi+1 {
			s[j-1].Hi = r[i].hi
		} else {
			s = s[0 : j+1]
			s[j] = unicode.Range32{
				Lo:     uint32(r[i].lo),
				Hi:     uint32(r[i].hi),
				Stride: 1,
			}
			j++
		}
	}
	return s
}

const format = "\t\t{0x%04x, 0x%04x, %d},\n"

func printRange(lo, hi, stride uint32, size int, count *int) (int, *int) {
	if size == 16 && hi >= 1<<16 {
		if lo < 1<<16 {
			if lo+stride != hi {
				logger.Fatalf("unexpected straddle: %U %U %d", lo, hi, stride)
			}
			// No range contains U+FFFF as an instance, so split
			// the range into two entries. That way we can maintain
			// the invariant that R32 contains only >= 1<<16.
			fmt.Printf(format, lo, lo, 1)
			lo = hi
			stride = 1
			*count++
		}
		fmt.Print("\t},\n")
		fmt.Print("\tR32: []Range32{\n")
		size = 32
		count = &range32Count
	}
	fmt.Printf(format, lo, hi, stride)
	*count++
	return size, count
}

func findLatinOffset(ranges []unicode.Range32) int {
	i := 0
	for i < len(ranges) && ranges[i].Hi <= unicode.MaxLatin1 {
		i++
	}
	return i
}

var range16Count = 0 // Number of entries in the 16-bit range tables.
var range32Count = 0 // Number of entries in the 32-bit range tables.

func printSizes() {
	fmt.Printf("// Range entries: %d 16-bit, %d 32-bit, %d total.\n", range16Count, range32Count, range16Count+range32Count)
	range16Bytes := range16Count * 3 * 2
	range32Bytes := range32Count * 3 * 4
	fmt.Printf("// Range bytes: %d 16-bit, %d 32-bit, %d total.\n", range16Bytes, range32Bytes, range16Bytes+range32Bytes)
}
