package blobproc

import (
	"os"
	"path"
	"testing"
)

func TestShardedPath(t *testing.T) {
	name := t.TempDir()
	svc := WebSpoolService{
		Dir: name,
	}
	var cases = []struct {
		about    string
		filename string
		create   bool
		result   string
		err      error
	}{
		{
			about:    "empty string",
			filename: "",
			create:   false,
			result:   "",
			err:      errShortName,
		},
		{
			about:    "short string",
			filename: "123",
			create:   false,
			result:   "",
			err:      errShortName,
		},
		{
			about:    "digest",
			filename: "34fc7a11cb38cf4911763696a41698c68e5ddbbe",
			create:   false,
			result:   path.Join(name, "/34/fc/7a11cb38cf4911763696a41698c68e5ddbbe"),
			err:      nil,
		},
		{
			about:    "digest",
			filename: "34fc7a11cb38cf4911763696a41698c68e5ddbbe.tei.xml",
			create:   false,
			result:   path.Join(name, "/34/fc/7a11cb38cf4911763696a41698c68e5ddbbe.tei.xml"),
			err:      nil,
		},
	}
	for _, c := range cases {
		result, err := svc.shardedPath(c.filename, c.create)
		if result != c.result {
			t.Fatalf("[%s] got %v, want suffix %v", c.about, result, c.result)
		}
		if err != c.err {
			t.Fatalf("[%s] got %v, want %v", c.about, err, c.err)
		}
		if err == nil {
			if c.create {
				if _, err := os.Stat(path.Dir(c.result)); os.IsNotExist(err) {
					t.Fatalf("expected dir: %v", path.Dir(c.result))
				}
			} else {
				if _, err := os.Stat(path.Dir(c.result)); err == nil {
					t.Fatalf("did not expect dir: %v", path.Dir(c.result))
				}
			}
		}
	}
}
