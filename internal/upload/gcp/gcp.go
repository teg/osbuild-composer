package gcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
)

func Upload(filename, bucket, object string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Open local file.
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	// Upload an object with storage.Writer.
	wc := client.Bucket(bucket).Object(object).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}
	return nil
}
