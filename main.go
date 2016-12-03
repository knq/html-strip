package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	tagPrefix = `<!-- ___XXX_%s_%d___ `
	tagSuffix = ` -->`
)

var (
	flagHidden     = flag.String("h", "HTML_STRIP", "hidden comment name")
	flagStrip      = flag.String("s", `script,noscript,link[rel="preload"][as="style"]`, "elements to strip")
	flagIgnoreTags = flag.String("i", `{%,%}`, "special tags to ignore")
)

func main() {
	var err error

	flag.Parse()

	a := flag.Args()
	if len(a) != 1 {
		fmt.Fprintf(os.Stderr, "error: please specify exactly one file to operate on\n")
		os.Exit(1)
	}

	// read file
	buf, err := ioutil.ReadFile(a[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// check tags
	tags := strings.Split(*flagIgnoreTags, ",")
	if len(tags)%2 == 1 {
		fmt.Fprintf(os.Stderr, "error: ignore tags must be in pairs\n", err)
		os.Exit(1)
	}

	// process tags
	for i := 0; i < len(tags); i += 2 {
		st := strings.TrimSpace(tags[i])
		et := strings.TrimSpace(tags[i+1])

		if st == "" || et == "" {
			fmt.Fprintf(os.Stderr, "error: invalid tags '%s,%s'", tags[i], tags[i+1])
			os.Exit(1)
		}

		prefix := fmt.Sprintf(tagPrefix, *flagHidden, i/2)
		re := regexp.MustCompile(regexp.QuoteMeta(st) + `[^` + regexp.QuoteMeta(et[:1]) + `]+` + regexp.QuoteMeta(et))
		buf = re.ReplaceAllFunc(buf, func(s []byte) []byte {
			sb := new(bytes.Buffer)
			sb.WriteString(prefix)
			sb.WriteString(base64.StdEncoding.EncodeToString(s))
			sb.WriteString(tagSuffix)
			return sb.Bytes()
		})
	}

	// load document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(buf)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// strip elements
	for _, s := range strings.Split(*flagStrip, ",") {
		doc.Find(strings.TrimSpace(s)).Each(func(i int, s *goquery.Selection) {
			s.Remove()
		})
	}

	// generate doc
	htmlstr, err := doc.Html()
	if err != nil {
		log.Fatalln(err)
	}

	// replace tags
	for i := 0; i < len(tags); i += 2 {
		prefix := fmt.Sprintf(tagPrefix, *flagHidden, i/2)
		re := regexp.MustCompile(prefix + `([^ ]+)` + tagSuffix)
		htmlstr = re.ReplaceAllStringFunc(htmlstr, func(s string) string {
			b, err := base64.StdEncoding.DecodeString(s[len(prefix) : len(s)-len(tagSuffix)])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: could not decode hidden tag: %v", err)
				os.Exit(1)
			}
			return string(b)
		})

		pf := html.EscapeString(prefix)
		sf := html.EscapeString(tagSuffix)
		re2 := regexp.MustCompile(regexp.QuoteMeta(pf) + `([^ ]+)` + regexp.QuoteMeta(sf))
		htmlstr = re2.ReplaceAllStringFunc(htmlstr, func(s string) string {
			b, err := base64.StdEncoding.DecodeString(s[len(pf) : len(s)-len(sf)])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: could not decode hidden tag: %v", err)
				os.Exit(1)
			}
			return string(b)
		})
	}

	os.Stdout.WriteString(htmlstr)
}
