package connector

import (
	"bytes"
	"fmt"
	"time"

	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
)

// This implementation of Emitter is used to store files from a Kinesis stream in S3. The use of
// this struct requires the configuration of an S3 bucket/endpoint. When the buffer is full, this
// struct's Emit method adds the contents of the buffer to S3 as one file. The filename is generated
// from the first and last sequence numbers of the records contained in that file separated by a
// dash. This struct requires the configuration of an S3 bucket and endpoint.
type S3Emitter struct {
	S3Bucket string
}

// Generates a file name based on the First and Last sequence numbers from the buffer. The current
// UTC date (YYYY-MM-DD) is base of the path to logically group days of batches.
func (e S3Emitter) S3FileName(firstSeq string, lastSeq string) string {
	date := time.Now().UTC().Format("2006-01-02")
	return fmt.Sprintf("/%v/%v-%v.txt", date, firstSeq, lastSeq)
}

// Invoked when the buffer is full. This method emits the set of filtered records.
func (e S3Emitter) Emit(buf Buffer) {
	auth, _ := aws.EnvAuth()
	s3Con := s3.New(auth, aws.USEast)
	bucket := s3Con.Bucket(e.S3Bucket)
	s3File := e.S3FileName(buf.FirstSequenceNumber(), buf.LastSequenceNumber())

	var buffer bytes.Buffer

	for _, r := range buf.Records() {
		buffer.WriteString(r.ToString())
	}

	err := bucket.Put(s3File, buffer.Bytes(), "text/plain", s3.Private, s3.Options{})

	if err != nil {
		fmt.Printf("Error occured while uploding to S3: %v\n", err)
	} else {
		fmt.Printf("Emitted %v records to S3 in s3://%v%v\n", buf.NumRecordsInBuffer(), e.S3Bucket, s3File)
	}
}