package main

import (
	"testing"
)

func TestExtractS3Uri(t *testing.T) {
	got := extractS3Uri("upload: www/index.html to s3://mybucketname/index.html")
	if got != "s3://mybucketname/index.html" {
		t.Errorf("ExtractS3Uri = %s; want s3://mybucketname/index.html", got)
	}
}

func TestExtractS3UriEmpty(t *testing.T) {
	got := extractS3Uri("upload: www/index.html")
	if got != "" {
		t.Errorf("ExtractS3Uri = %s; want empty string", got)
	}
}
