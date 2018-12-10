package main

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"

    "fmt"
    "os"
)

// TODO fill these in!
const (
    S3Region = "eu-central-1"
    S3Bucket = "pl.rrj.icm-polly"
)

func main() {
    if len(os.Args) != 2 {
        exitErrorf("Item name required\nUsage: %s item_name",
        os.Args[0])
    }

    item := os.Args[1]

    file, err := os.Create(item)
    if err != nil {
        exitErrorf("Unable to open file %q, %v", err)
    }

    defer file.Close()

    // Initialize a session that the SDK will use to load
    // credentials from the shared credentials file ~/.aws/credentials.
    sess, _ := session.NewSession(&aws.Config{
        Region: aws.String(S3Region)},
    )

    downloader := s3manager.NewDownloader(sess)

    numBytes, err := downloader.Download(file,
    &s3.GetObjectInput{
        Bucket: aws.String(S3Region),
        Key:    aws.String(item),
    })
    if err != nil {
        exitErrorf("Unable to download item %q, %v", item, err)
    }

    fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
}

func exitErrorf(msg string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, msg+"\n", args...)
    os.Exit(1)
}