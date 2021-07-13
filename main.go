package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-enry/go-enry/v2"
	sitter "github.com/smacker/go-tree-sitter"
)

var (
	noFileNames bool
	queryFile   string
	lang        string
)

func init() {
	flag.BoolVar(&noFileNames, "q", false, `"quiet" mode excludes file names from output`)
	flag.StringVar(&queryFile, "f", "", "query can be extracted from filepath")
	flag.StringVar(&lang, "lang", "", "language can be given by user")
	flag.Parse()
}

func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func treequery(path string, queryName string) {
	absPath, err := filepath.Abs(path)
	handleErr(err)

	contents, err := ioutil.ReadFile(absPath)
	handleErr(err)

	if len(lang) == 0 {
		lang = enry.GetLanguage(absPath, contents)
	}
	if lang == "" {
		handleErr(errors.New("language could not be detected"))
	}

	language, query := getTSLanguageFromEnry(lang)
	if len(queryFile) != 0 {
		queryContent, err := ioutil.ReadFile(queryFile)
		handleErr(err)
		query = string(queryContent)
	}
	fmt.Println(query)

	if language == nil {
		handleErr(fmt.Errorf("no parser for: %s", lang))
	}

	parser := sitter.NewParser()
	parser.SetLanguage(language)

	tree := parser.Parse(nil, contents)
	n := tree.RootNode()

	if query == "" {
		// fmt.Println(absPath)
		// fmt.Println(n.Content(contents))
		fmt.Println(n)
	}

	q, err := sitter.NewQuery([]byte(query), language)
	if err != nil {
		handleErr(fmt.Errorf("problem with query: %w", err))
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(q, n)

	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}

		// fmt.Println(q.CaptureNameForId(m.ID))
		for _, c := range m.Captures {
			if q.CaptureNameForId(c.Index) == queryName {
				if !noFileNames {
					fmt.Printf("%s:%d:%d\n", absPath, c.Node.StartPoint().Row+1, c.Node.StartPoint().Column+1)
				}
				fmt.Println(c.Node.Content(contents))
			}
		}
	}
}

func main() {
	path := flag.Arg(0)
	queryName := flag.Arg(1)

	pathInfo, err := os.Stat(path)
	handleErr(err)

	if pathInfo.IsDir() {
		entries, err := os.ReadDir(path)
		handleErr(err)
		for _, entry := range entries {
			if !entry.IsDir() {
				file := path + "/" + entry.Name()
				treequery(file, queryName)
			} else {
				fmt.Println("directory found")
			}
		}
	} else {
		treequery(path, queryName)
	}
}
