package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// r2Client builds an S3 client for a Cloudflare R2 destination. R2 has no
// regions, so the fixed region "auto" is used, and the bucket is addressed in
// the path so a custom endpoint works too.
func r2Client(d *Destination) (*minio.Client, error) {
	host, secure := d.AccountID+".r2.cloudflarestorage.com", true
	if d.Endpoint != "" {
		u, err := url.Parse(d.Endpoint)
		if err != nil {
			return nil, err
		}
		host, secure = u.Host, u.Scheme == "https"
	}
	return minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(d.AccessKeyID, d.SecretAccessKey, ""),
		Secure:       secure,
		Region:       "auto",
		BucketLookup: minio.BucketLookupPath,
	})
}

// r2Prefix is the key prefix backups are stored under, with a trailing slash,
// or empty for the bucket root.
func r2Prefix(d *Destination) string {
	p := strings.Trim(d.FolderPath, "/")
	if p == "" {
		return ""
	}
	return p + "/"
}

// uploadR2 stores the backup as one object. Large files are sent in parts by
// the client. Like the file server upload it is aborted when no data has been
// sent for uploadStallTimeout.
func uploadR2(ctx context.Context, d *Destination, file string) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	client, err := r2Client(d)
	if err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}

	progress := &progressReader{r: f}
	progress.touch()
	go watchStall(ctx, cancel, progress)
	key := r2Prefix(d) + filepath.Base(file)
	_, err = client.PutObject(ctx, d.Bucket, key, progress, st.Size(), minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		if cause := context.Cause(ctx); errors.Is(cause, errUploadStalled) {
			return fmt.Errorf("%w for %s", cause, uploadStallTimeout)
		}
		return fmt.Errorf("upload to r2 bucket %s: %w", d.Bucket, err)
	}
	return nil
}

// pruneR2 deletes dbName's backups under the destination's prefix beyond the
// newest keepLast or older than keepDays, with the same rules as the local
// folder.
func pruneR2(ctx context.Context, d *Destination, dbName string) error {
	client, err := r2Client(d)
	if err != nil {
		return err
	}
	prefix := r2Prefix(d)
	var names []string
	for obj := range client.ListObjects(ctx, d.Bucket, minio.ListObjectsOptions{Prefix: prefix}) {
		if obj.Err != nil {
			return fmt.Errorf("list r2 bucket %s: %w", d.Bucket, obj.Err)
		}
		// Only objects directly under the prefix; sub-folders are not ours.
		if name := strings.TrimPrefix(obj.Key, prefix); name != "" && !strings.Contains(name, "/") {
			names = append(names, name)
		}
	}
	var errs []error
	for _, name := range pruneCandidates(names, dbName, d.KeepLast, d.KeepDays) {
		if err := client.RemoveObject(ctx, d.Bucket, path.Join(prefix, name), minio.RemoveObjectOptions{}); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
