package main


import (
    "bytes"
    "crypto/sha1"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "io"
    "io/ioutil"
    "strings"
    "time"

    "github.com/PuerkitoBio/goquery"
    "golang.org/x/net/html/charset"
    "github.com/dmulholland/mp3lib"
    "mvdan.cc/xurls/v2"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/polly"
    "github.com/aws/aws-sdk-go/service/s3"
)
// TODO fill these in!
const (
    S3Region = "eu-central-1"
    S3Bucket = "pl.rrj.icm-polly"
    LinkExpiration = 180
    Voice = "Maja"
)

/* Paragraph TODO */
type paragraph struct {
    Iter int
    Text string
    AccessURL string
}
/* article TODO */
type article struct {
    Text string
    Hash string
    Paragraphs [] paragraph
    Urls [] string
    AccessURL string
    AudioURL string
    Timestamp int64
    Valid int64
}

func main() {

    var k article
    var link string

    body, err := fetchUtf8Bytes("http://www.meteo.pl/komentarze/index1.php")
    if err != nil {
        log.Println("Error: ", err)
    }
    // Create a goquery document from the HTTP response
    document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
    if err != nil {
        log.Println("Error loading HTTP response body. ", err)
    }
    // get 3rd div
    komentarz := document.Find("div").Eq(3).Text()
    //remove empty paragraphs
    komentarz = strings.Replace(komentarz, "\n\n", "\n", -1)
    komentarz = strings.Replace(komentarz, "\n ", "\n", -1)
    komentarz = strings.Replace(komentarz, " \n", "\n", -1)
    k.Text = komentarz
    // TODO - make SSML
    log.Println(k.Text)
    // compute signature
    h := sha1.New()
    h.Write([]byte(komentarz))
    k.Hash = fmt.Sprintf("%x", h.Sum(nil))
    log.Println(k.Hash)
    k.Timestamp = time.Now().Unix()
    k.Valid = LinkExpiration * 60 + k.Timestamp
    // extract urls
    k.Urls = xurls.Relaxed().FindAllString(komentarz, -1)
    log.Println(k.Urls)
    // Initialize AWS Session
    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))
    // Create Polly client
    svc := polly.New(sess)
    // Split into paragraphs
    paragraphs := strings.Split(komentarz, "\n")
    // create filename (prefix + hash
    fn := fmt.Sprintf("icm_%s", k.Hash)
    // Upload komentarz
    err = addTextToS3(sess, komentarz, fn + ".txt")
    if err != nil {
        log.Print(err.Error())
    }
    link, err = getFileLink(sess, fn + ".txt")
    if err != nil {
        log.Print(err.Error())
    }
    k.AccessURL = link
    log.Println(k.AccessURL)
    // Convert each paragraph into speach with Polly and upload to S3
    j := 1
    merged := new(bytes.Buffer)
    var totalFrames uint32
    var totalBytes uint32
    // var totalFiles int
    // var firstBitRate int

    for _, p := range paragraphs {
        if len([]rune(p)) < 5 {
            log.Printf("Skipping -->%s<--", p)
            continue
        }
        log.Println(p)
        // Output to MP3 using voice Maja (PL)
        input := &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), Text: aws.String(p), VoiceId: aws.String(Voice)}
        output, err := svc.SynthesizeSpeech(input)
        if err != nil {
            log.Println("Got error when calling SynthesizeSpeech:")
            log.Print(err.Error())
            continue
        }
        // merge separate paragraph streams in one single file
        isFirstFrame := true
        for {
            // Read the next frame from the input file.
            frame := mp3lib.NextFrame(output.AudioStream)
            if frame == nil {
                break
            }
            // Skip the first frame if it's a VBR header.
            if isFirstFrame {
                isFirstFrame = false
                if mp3lib.IsXingHeader(frame) || mp3lib.IsVbriHeader(frame) {
                    continue
                }
            }
            // Write the frame to the output stream
            _, err = merged.Write(frame.RawBytes)
            if err != nil {
                log.Println(err.Error())
            }
            totalFrames++
            totalBytes += uint32(len(frame.RawBytes))
        } // for (merging files)
        // TODO - we don't need paragraph's mp3 ? Anyway they are empty now
        // create filename (prefix, hash, part number + extension)
        // fna := fn + "_" + strconv.Itoa(i) + ".mp3"
        // // Upload audio file
        // err = addAudiostreamToS3(sess, output.AudioStream, fna)
        // if err != nil {
        //     log.Print(err.Error())
        //     continue
        // }
        // link, err = getFileLink(sess, fna)
        // if err != nil {
        //     log.Print(err.Error())
        //     continue
        // }
        // log.Println(link)
        k.Paragraphs = append(k.Paragraphs, paragraph{Iter: j, Text: p, AccessURL: link})
        j++
    }
    // save merged steram to S3
    err = addAudiostreamToS3(sess, ioutil.NopCloser(merged), fn + ".mp3")
    if err != nil {
        log.Println(err.Error())
    }
    link, err = getFileLink(sess, fn + ".mp3")
    if err != nil {
        log.Print(err.Error())
    }
    k.AudioURL = link
    log.Println(k.AudioURL)

    // Prepare JSON
    js, err := json.MarshalIndent(k, "  ", "    ")
    if err != nil {
        log.Println(err.Error())
    }
    log.Println(string(js))
}


// Save Audiostream generated by Polly to S3
func addAudiostreamToS3(s *session.Session, pollyStream io.ReadCloser, fileName string) error {

    buffer := streamToByte(pollyStream)
    size := int64(len(buffer))

    // Config settings: this is where you choose the bucket, filename, content-type etc.
    // of the file you're uploading.
    _, err := s3.New(s).PutObject(&s3.PutObjectInput{
        Bucket:               aws.String(S3Bucket),
        Key:                  aws.String(fileName),
        ACL:                  aws.String("private"),
        Body:                 bytes.NewReader(buffer),
        ContentLength:        aws.Int64(size),
        ContentType:          aws.String(http.DetectContentType(buffer)),
        ContentDisposition:   aws.String("attachment"),
        ServerSideEncryption: aws.String("AES256"),
    })
    return err
}

// Save text (string) to S3
func addTextToS3(s *session.Session, comm string, fileName string) error {

    buffer := []byte(comm)
    size := int64(len(buffer))

    // Config settings: this is where you choose the bucket, filename, content-type etc.
    // of the file you're uploading.
    _, err := s3.New(s).PutObject(&s3.PutObjectInput{
        Bucket:               aws.String(S3Bucket),
        Key:                  aws.String(fileName),
        ACL:                  aws.String("private"),
        Body:                 bytes.NewReader(buffer),
        ContentLength:        aws.Int64(size),
        ContentType:          aws.String(http.DetectContentType(buffer)),
        ContentDisposition:   aws.String("attachment"),
        ServerSideEncryption: aws.String("AES256"),
    })
    return err
}

// Get public time-limited link to private object in S3 bucket
// if what you want is just the URL of a public access object you can build the URL yourself:
// https://<region>.amazonaws.com/<bucket-name>/<key>
func getFileLink(s *session.Session, fileName string) (string, error) {

    req, _ := s3.New(s).GetObjectRequest(&s3.GetObjectInput{
        Bucket:             aws.String(S3Bucket),
        Key:                aws.String(fileName),
    })
    url, err := req.Presign(LinkExpiration * time.Minute) // Set link expiration time

    return url, err
}

// ICM is using ISO-8859-2 which must be converted to UTF
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

// convert AudioStreams to []byte 
func streamToByte(stream io.Reader) []byte {
    buf := new(bytes.Buffer)
    buf.ReadFrom(stream)
    return buf.Bytes()
}
