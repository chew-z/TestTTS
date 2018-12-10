package main

import (
    "bytes"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
)

// TODO fill these in!
const (
    S3Region = "eu-central-1"
    S3Bucket = "pl.rrj.icm-polly"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("You must supply file name")
        os.Exit(1)
    }

    // The name of the text file to convert to MP3
    fileName := os.Args[1]
 
    // Create a single AWS session (we can re use this if we're uploading many files)
    s, err := session.NewSession(&aws.Config{Region: aws.String(S3Region)})
    if err != nil {
        log.Fatal(err)
    }

    // Upload
    err = AddFileToS3(s, fileName)
    if err != nil {
        log.Fatal(err)
    }
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, fileDir string) error {

    // Open the file for use
    file, err := os.Open(fileDir)
    if err != nil {
        return err
    }
    defer file.Close()

    // Get file size and read the file content into a buffer
    fileInfo, _ := file.Stat()
    var size int64 = fileInfo.Size()
    buffer := make([]byte, size)
    file.Read(buffer)

    // Config settings: this is where you choose the bucket, filename, content-type etc.
    // of the file you're uploading.
    _, err = s3.New(s).PutObject(&s3.PutObjectInput{
        Bucket:               aws.String(S3Bucket),
        Key:                  aws.String(fileDir),
        ACL:                  aws.String("private"),
        Body:                 bytes.NewReader(buffer),
        ContentLength:        aws.Int64(size),
        ContentType:          aws.String(http.DetectContentType(buffer)),
        ContentDisposition:   aws.String("attachment"),
        ServerSideEncryption: aws.String("AES256"),
    })
    return err
}