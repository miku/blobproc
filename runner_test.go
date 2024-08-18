package blobproc

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/miku/blobproc/fileutils"
	"github.com/miku/grobidclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestBlobprocRoundtrip starts grobid and minio and will process a PDF file.
func TestBlobprocRoundtrip(t *testing.T) {
	skipNoDocker(t)
	if testing.Short() {
		t.Skip("skipping testcontainer based tests in short mode")
	}
	ctx := context.Background()
	// MINIO
	req := testcontainers.ContainerRequest{
		Image: "quay.io/minio/minio:latest",
		ExposedPorts: []string{
			"9000/tcp",
			"9001/tcp",
		},
		WaitingFor: wait.ForListeningPort("9000/tcp"),
		Cmd: []string{
			"minio",
			"server",
			"/tmp",
		},
	}
	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("could not start minio: %s", err)
	}
	defer func() {
		if err := minioC.Terminate(ctx); err != nil {
			t.Fatalf("could not stop minio: %s", err)
		}
	}()
	// GROBID
	req = testcontainers.ContainerRequest{
		Image:        "grobid/grobid:0.8.0",
		ExposedPorts: []string{"8070/tcp"},
		WaitingFor:   wait.ForListeningPort("8070/tcp"),
	}
	grobidC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	defer func() {
		if err := grobidC.Terminate(ctx); err != nil {
			t.Fatalf("Could not stop grobid: %s", err)
		}
	}()
	t.Logf("started two containers: %v, %v", minioC, grobidC)
	ip, port, err := containerHostPort(ctx, grobidC, "8070")
	if err != nil {
		t.Fatalf("testcontainers: %v", err)
	}
	grobidHostport := fmt.Sprintf("http://%s:%s", ip, port)
	ip, port, err = containerHostPort(ctx, minioC, "9000")
	if err != nil {
		t.Fatalf("testcontainers: %v", err)
	}
	minioHostport := fmt.Sprintf("%s:%s", ip, port)
	t.Logf("found minio hostport at: %v", minioHostport)
	dir, err := os.MkdirTemp("", "blobproc-test-spool-*")
	if err != nil {
		t.Fatalf("temp dir failed: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	t.Logf("setup spool dir at %v", dir)
	dst := path.Join(dir, "1.pdf")
	if err := fileutils.CopyFile(dst, "testdata/pdf/1906.02444.pdf"); err != nil {
		t.Fatalf("spool dir copy failed: %v", err)
	}
	s3wrapper, err := NewWrapS3(minioHostport, &WrapS3Options{
		AccessKey:     "minioadmin",
		SecretKey:     "minioadmin",
		DefaultBucket: "sandcrawler",
		UseSSL:        false,
	})
	if err != nil {
		t.Fatalf("s3 failed: %v", err)
	}
	grobid := grobidclient.New(grobidHostport)
	runner := &Runner{
		Grobid:            grobid,
		MaxGrobidFileSize: 256 * 1024 * 1024,
		ConsolidateMode:   false,
		S3Wrapper:         s3wrapper,
	}
	sha1hex, err := runner.RunGrobid(dst)
	if err != nil {
		t.Fatalf("run grobid: got %v, want nil", err)
	}
	b, err := s3wrapper.GetBlob(context.TODO(), &BlobRequestOptions{
		Folder:  "grobid",
		SHA1Hex: sha1hex,
		Ext:     ".tei.xml",
		Prefix:  "",
		Bucket:  "sandcrawler",
	})
	if err != nil {
		t.Fatalf("could not retrieve result: %v", err)
	}
	t.Logf("parse result: %v", string(b))
	if err := runner.RunPdfToText(dst); err != nil {
		t.Fatalf("failed to extract text: %v", err)
	}
	t.Logf("roundtrip completed")
}

// containerHostPort return the ip and port as string for a given testcontainer.
func containerHostPort(ctx context.Context, c testcontainers.Container, mappedPort string) (ip, port string, err error) {
	ip, err = c.Host(ctx)
	if err != nil {
		return
	}
	p, err := c.MappedPort(ctx, nat.Port(mappedPort))
	if err != nil {
		return "", "", err
	}
	port = strings.Split(string(p), "/")[0]
	return ip, port, nil
}
