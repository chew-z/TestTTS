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
-g, --goquery <query>   goquery expression.
-o, --out <path>        Output filename. Defaults to 'output.txt'.

Flags:
-d, --document          Print nodes of entire document.
-t, --tree              Print nodes of selected element.
-l, --link              Extract links from document.
-h, --help              Display this help text and exit.
-v, --version           Display the application's version number and exit.
`, filepath.Base(os.Args[0]))

func main() {
	url := "http://www.meteo.pl/komentarze/index1.php"
	gq := ""
	fn := "output.txt"
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
		fmt.Println("---")
		document.Find("*").Each(func(_ int, node *goquery.Selection) {
			fmt.Println("---")
			fmt.Println(node.Html())
			fmt.Println("---")
		})
		fmt.Println("---")
		os.Exit(0)
	}
	// All elements matching query - HTML and text
	if tree {
		fmt.Println("---")
		document.Find(gq).Each(func(_ int, node *goquery.Selection) {
			fmt.Println("---")
			fmt.Println(node.Html())
			fmt.Println(node.Text())
			fmt.Println("---")
		})
		fmt.Println("---")
		os.Exit(0)
	}
	// extract urls from elements matching query
	if links {
		document.Find(gq).Each(func(_ int, node *goquery.Selection) {
			fmt.Println(xurls.Relaxed().FindAllString(node.Text(), -1))
		})
		os.Exit(0)
	}
	element := document.Find(gq)
	if n > -1 {
		element = element.Eq(n)
	}
	text := element.Text()
	fmt.Println(text)

	// Save to file
	err = ioutil.WriteFile(fn, []byte(text), 0666)
	if err != nil {
		log.Fatal(err)
	}

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
