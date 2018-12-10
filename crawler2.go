package main

import (
    "bytes"
    "crypto/sha1"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "strings"

    "github.com/PuerkitoBio/goquery"
    "golang.org/x/net/html/charset"
    "github.com/mvdan/xurls"
)

func main() {

    body, err := fetchUtf8Bytes("http://www.meteo.pl/komentarze/index1.php")
    if err != nil {
        fmt.Println("Error: ", err)
    }

    r := bytes.NewReader(body)

    // Create a goquery document from the HTTP response
    document, err := goquery.NewDocumentFromReader(r)
    if err != nil {
        log.Fatal("Error loading HTTP response body. ", err)
    }

    komentarz := document.Find("div").Eq(3).Text()
    //remove empty paragraphs
    komentarz = strings.Replace(komentarz, "\n\n", "\n", -1)
    fmt.Println(komentarz)
    // extract urls
    fmt.Println(xurls.Relaxed().FindAllString(komentarz, -1))
    // compute signature
    h := sha1.New()
    h.Write([]byte(komentarz))
    bs := h.Sum(nil)
    fmt.Printf("%x\n", bs)

    fn := fmt.Sprintf("icm_%x", bs) + ".txt"
    err = ioutil.WriteFile(fn, []byte(komentarz), 0666)
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

