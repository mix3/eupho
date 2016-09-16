package eupho

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func newTempFiles(files map[string]string) (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	for name, content := range files {
		tmpFn := filepath.Join(dir, name)
		if err := ioutil.WriteFile(tmpFn, []byte(content), 0644); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}
