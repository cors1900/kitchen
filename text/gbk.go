package text

import (
	"bufio"
	"os"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func ReadFileWithGBKEncoding(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	enc := simplifiedchinese.GBK
	r := transform.NewReader(f, enc.NewDecoder())

	// Read converted UTF-8 from `r` as needed.
	// As an example we'll read line-by-line showing what was read:
	var utf8 []byte
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		utf8 = append(utf8, sc.Bytes()...)
	}
	if err = sc.Err(); err != nil {
		return nil, err
	}

	if err = f.Close(); err != nil {
		return nil, err
	}
	return utf8, nil
}
