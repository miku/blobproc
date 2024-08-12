package blobproc

import "testing"

func TestBlobPath(t *testing.T) {
	var cases = []struct {
		about   string
		folder  string
		sha1hex string
		ext     string
		prefix  string
		result  string
	}{
		{
			about:   "empty",
			folder:  "",
			sha1hex: "4e1243bd22c66e76c2ba9eddc1f91394e57f9f83",
			ext:     "",
			prefix:  "",
			result:  "/4e/12/4e1243bd22c66e76c2ba9eddc1f91394e57f9f83",
		},
		{
			about:   "folder",
			folder:  "images",
			sha1hex: "4e1243bd22c66e76c2ba9eddc1f91394e57f9f83",
			ext:     "",
			prefix:  "",
			result:  "images/4e/12/4e1243bd22c66e76c2ba9eddc1f91394e57f9f83",
		},
		{
			about:   "folder, ext",
			folder:  "images",
			sha1hex: "4e1243bd22c66e76c2ba9eddc1f91394e57f9f83",
			ext:     "xml",
			prefix:  "",
			result:  "images/4e/12/4e1243bd22c66e76c2ba9eddc1f91394e57f9f83.xml",
		},
		{
			about:   "folder, ext, prefix",
			folder:  "images",
			sha1hex: "4e1243bd22c66e76c2ba9eddc1f91394e57f9f83",
			ext:     "xml",
			prefix:  "dev-",
			result:  "dev-images/4e/12/4e1243bd22c66e76c2ba9eddc1f91394e57f9f83.xml",
		},
	}
	for _, c := range cases {
		result := blobPath(c.folder, c.sha1hex, c.ext, c.prefix)
		if result != c.result {
			t.Fatalf("[%s] got %v, want %v", c.about, result, c.result)
		}
	}
}
