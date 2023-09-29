package targets

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Test mg.Namespace

// All runs all tests.
func (Test) All() error {
	return sh.RunWithV(map[string]string{
		"CGO_ENABLED": "1",
	}, mg.GoCmd(), "test", "-race", "./...")
}

// Short runs all tests in short mode.
func (Test) Short() error {
	return sh.RunWithV(map[string]string{
		"CGO_ENABLED": "1",
	}, mg.GoCmd(), "test", "-race", "-short", "./...")
}

// Cover runs all tests with coverage analysis.
func (Test) Cover() error {
	err := sh.RunWithV(map[string]string{
		"CGO_ENABLED": "1",
	}, mg.GoCmd(), "test",
		"-race",
		"-cover",
		"-coverprofile=cover.out",
		"-coverpkg=./...",
		"./...",
	)
	if err != nil {
		return err
	}
	fmt.Print("processing coverage report... ")
	defer fmt.Println("done.")
	return filterCoverage("cover.out", []string{
		"**/*.pb.go",
		"**/*.pb*.go",
		"**/zz_*.go",
	})
}

func filterCoverage(report string, patterns []string) error {
	f, err := os.Open(report)
	if err != nil {
		return err
	}
	defer f.Close()

	tempFile := fmt.Sprintf(".%s.tmp", report)
	tf, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	patternIndex := 0
	scan := bufio.NewScanner(f)
	scan.Scan() // mode line
	_, _ = tf.WriteString(scan.Text() + "\n")
LINES:
	for scan.Scan() {
		line := scan.Text()
		filename, _, _ := strings.Cut(line, ":")
		var j int
		for i := patternIndex; j < len(patterns); i = (i + 1) % len(patterns) {
			match, _ := doublestar.Match(patterns[i], filename)
			if match {
				continue LINES
			}
			j++
		}
		_, _ = tf.WriteString(line + "\n")
	}
	if err := scan.Err(); err != nil {
		return err
	}
	_ = tf.Close()

	return os.Rename(tempFile, report)
}
