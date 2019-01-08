/*
Quick and dirty crawler for peeking into HTML elements using goquery

*/

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dmulholland/janus-go/janus"
	"golang.org/x/net/html/charset"
	"mvdan.cc/xurls/v2"
)

const version = "0.2.1"

var helptext = fmt.Sprintf(`
Usage: %s [FLAGS] [OPTIONS] [URL]

Quick and dirty crawler for peeking into HTML elements using goquery

Arguments: URL          URL to scrap.

Options:
-g, --goquery <query>   goquery expression (a'la jQuery #, . ).
-o, --out <path>        Output filename. Defaults to 'output.txt'.
-n <int>                If multiple elements match select n-th. (-1) for all.

Flags:
-d, --document          Print nodes of entire document.
-t, --tree              Print nodes of selected element.
-l, --link              Extract links from document.
-p, process             Process result with internal function (clean, regex).
-h, --help              Display this help text and exit.
-v, --version           Display the application's version number and exit.
`, filepath.Base(os.Args[0]))

func main() {
	url := "http://www.meteo.pl/komentarze/index1.php"
	gq := ""
	fn := "output.txt"
	var sb strings.Builder
	// Parse the command line arguments.
	parser := janus.NewParser()
	parser.Helptext = helptext
	parser.Version = version
	parser.NewString("out o", "output.txt")
	parser.NewString("goquery g", "body")
	parser.NewInt("n")
	parser.NewFlag("tree t")
	parser.NewFlag("document d")
	parser.NewFlag("links l")
	parser.NewFlag("process p")
	parser.Parse()

	// Get url
	var urls []string
	n := -1
	if parser.HasArgs() {
		urls = parser.GetArgs()
		url = urls[0]
	} else {
		log.Printf("Warning: no URL argument given, using default %s", url)
		// os.Exit(1)
	}
	log.Println(url)
	// Get goquery expression
	if parser.Found("goquery") {
		gq = parser.GetString("goquery")
	}
	if parser.Found("n") {
		n = parser.GetInt("n")
	}
	// Get output filename
	fn = parser.GetString("out")
	tree := parser.GetFlag("tree")
	dee := parser.GetFlag("document")
	links := parser.GetFlag("links")
	proc := parser.GetFlag("process")

	body, err := fetchUtf8Bytes(url)
	if err != nil {
		log.Println("Error: ", err)
	}
	r := bytes.NewReader(body)
	// Create a goquery document from the HTTP response
	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}
	// Entire document tree
	if dee {
		var nHtml string
		sb.WriteString("---")
		document.Find("*").Each(func(_ int, node *goquery.Selection) {
			sb.WriteString("---")
			nHtml, _ = node.Html()
			sb.WriteString(nHtml)
			sb.WriteString("---")
		})
		sb.WriteString("---")
		saveTofile(fn, sb.String())
		fmt.Println(sb.String())
		os.Exit(0)
	}
	// All elements matching query - HTML and text
	if tree {
		var nHtml string
		sb.WriteString("---")
		document.Find(gq).Each(func(_ int, node *goquery.Selection) {
			sb.WriteString("---")
			nHtml, _ = node.Html()
			sb.WriteString(nHtml)
			sb.WriteString(node.Text())
			sb.WriteString("---")
		})
		sb.WriteString("---")
		saveTofile(fn, sb.String())
		fmt.Println(sb.String())
		os.Exit(0)
	}
	// extract urls from elements matching query
	if links {
		var links []string
		document.Find(gq).Each(func(_ int, node *goquery.Selection) {
			links = xurls.Relaxed().FindAllString(node.Text(), -1)
			for _, l := range links {
				sb.WriteString(l)
			}
		})
		saveTofile(fn, sb.String())
		fmt.Println(sb.String())
		os.Exit(0)
	}
	elements := document.Find(gq)
	if n > -1 {
		text := elements.Eq(n).Text()
		if proc {
			text = cleanText(text)
		}
		sb.WriteString(text)
	} else {
		elements.Each(func(i int, s *goquery.Selection) {
			if proc {
				// log.Printf("+>%s<+", s.Text())
				// log.Printf("=>%s<=", cleanText(s.Text()))
				sb.WriteString(cleanText(s.Text()))
			} else {
				sb.WriteString(s.Text())
			}
		})
	}
	paragraphs := strings.Split(sb.String(), "\n")
	sb.Reset()
	for _, p := range paragraphs {
		if len([]rune(p)) < 4 {
			continue
		}
		// log.Printf("-->%s<--", p)
		ssml := makeSSML(p)
		// log.Println(ssml)
		sb.WriteString(ssml)
		sb.WriteString("\n")
	} // Save to file
	saveTofile(fn, sb.String())
	fmt.Println(sb.String())
	os.Exit(0)
}

func cleanText(in string) string {
	var out string
	// https://github.com/google/re2/wiki/Syntax
	re1 := regexp.MustCompile(`(?m)[[:blank:]]{2,}`) // multiple whitespace
	re2 := regexp.MustCompile(`(»|zobacz więcej|Zobacz TOTERAZ|czytaj dalej)`)
	// dots followed by sentence not separated by space
	re3 := regexp.MustCompile(`(\.)([[:upper:]])`)
	out = re1.ReplaceAllString(in, "")
	out = re2.ReplaceAllString(out, "")
	out = re3.ReplaceAllString(out, "$1 $2")
	out = out + "\n"
	return out
}

// Make SSML from paragraph, short sentence
func makeSSML(in string) string {
	var sb strings.Builder
	sb.WriteString("<speak>")
	sb.WriteString("<amazon:auto-breaths frequency='low' volume='medium' duration='medium'>")
	sb.WriteString("<p>")
	sb.WriteString(in)
	sb.WriteString("</p>")
	sb.WriteString("</amazon:auto-breaths>")
	sb.WriteString("</speak>")

	return sb.String()
}

// ICM is using ISO-8859-2
func fetchUtf8Bytes(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	contentType := res.Header.Get("Content-Type") // Optional, better guessing
	utf8reader, err := charset.NewReader(res.Body, contentType)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(utf8reader)
}

func saveTofile(fn string, text string) {
	err := ioutil.WriteFile(fn, []byte(text), 0666)
	if err != nil {
		log.Fatal(err)
	}
}
