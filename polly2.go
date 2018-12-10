/*
Copyright 2010-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.

This file is licensed under the Apache License, Version 2.0 (the "License").
You may not use this file except in compliance with the License. A copy of
the License is located at

http://aws.amazon.com/apache2.0/

This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
*/

package main

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/polly"

    "bufio"
    "fmt"
    "io"
    "log"
    "os"
    "strings"
    "strconv"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("You must supply file name")
        os.Exit(1)
    }

    // The name of the text file to convert to MP3
    fileName := os.Args[1]
    names := strings.Split(fileName, ".")
    name := names[0]

    file, err := os.Open(fileName)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // Initialize a session that the SDK uses to load
    // credentials from the shared credentials file. (~/.aws/credentials).
    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))

    // Create Polly client
    svc := polly.New(sess)

    scanner := bufio.NewScanner(file)

    i := 0
    s := ""
    for scanner.Scan() {
        s = scanner.Text()
        if len(s) < 5 {
            continue
        }
        fmt.Println(s)
        // Output to MP3 using voice Maja (PL)
        input := &polly.SynthesizeSpeechInput{OutputFormat: aws.String("mp3"), Text: aws.String(s), VoiceId: aws.String("Maja")}

        output, err := svc.SynthesizeSpeech(input)
        if err != nil {
            fmt.Println("Got error calling SynthesizeSpeech:")
            fmt.Print(err.Error())
            os.Exit(1)
        }

        // Save as MP3
        mp3File := name + "_" + strconv.Itoa(i) + ".mp3"
        i++

        outFile, err := os.Create(mp3File)
        if err != nil {
            fmt.Println("Got error creating " + mp3File + ":")
            fmt.Print(err.Error())
            os.Exit(1)
        }

        defer outFile.Close()
        _, err = io.Copy(outFile, output.AudioStream)
        if err != nil {
            fmt.Println("Got error saving MP3:")
            fmt.Print(err.Error())
            os.Exit(1)
        }
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

}
