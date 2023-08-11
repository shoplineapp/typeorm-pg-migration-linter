package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type LintResult struct {
	SQL  string
	Ok   bool
	Lint string
}

func main() {
	app := &cli.App{
		Name:   "typeorm-migration-linter",
		Usage:  "tm-linter [paths], use comma to separate multiple paths",
		Action: handleAction(),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func handleAction() func(cCtx *cli.Context) error {
	return func(cCtx *cli.Context) error {
		paths := cCtx.Args().Get(0)
		return checkAllLint(paths)
	}
}

func checkAllLint(paths string) error {
	if paths == "" {
		return cli.Exit("empty path", -1)
	}

	var fileList []string

	pathList := strings.Split(paths, ",")

	for _, path := range pathList {
		stat, err := os.Stat(path)

		if err != nil {
			return cli.Exit(err, -1)
		}

		if stat.IsDir() {
			files := readFolder(path)
			fileList = append(fileList, files...)
		} else {
			fileList = append(fileList, path)
		}
	}

	fmt.Println("file to check:", strings.Join(fileList, ", "))

	contents, err := bulkReadFileContent(fileList)
	if err != nil {
		log.Fatal(err)
	}

	allSql := bulkFindQuery(contents)

	var pass []LintResult
	var fail []LintResult

	for _, sql := range allSql {
		result := runLint(sql)
		if result.Ok {
			pass = append(pass, result)
		} else {
			fail = append(fail, result)
		}
	}

	fmt.Println("pass:", len(pass))
	fmt.Println("fail:", len(fail))

	if len(fail) > 0 {
		fmt.Println("fail sql:")
		for _, result := range fail {
			fmt.Println(result.SQL)
			fmt.Println(result.Lint)
		}

		return cli.Exit("lint not pass", -1)
	}

	return nil
}

func readFolder(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	var pathInDir []string

	for _, e := range entries {
		pathInDir = append(pathInDir, path+"/"+e.Name())
	}

	return pathInDir
}

func bulkReadFileContent(paths []string) (content []string, err error) {
	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		content = append(content, string(bytes))
	}

	return
}

func bulkFindQuery(contents []string) (allSql []string) {
	for _, content := range contents {
		regex := regexp.MustCompile(`\.query\(\s*` + "`" + `([^` + "`" + `]+)` + "`")
		matches := regex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) >= 2 {
				allSql = append(allSql, match[1])
			}
		}
	}

	return
}

func runLint(sql string) LintResult {
	cmdString := fmt.Sprintf(`echo "%s" | squawk --exclude=ban-drop-column`, sql)

	cmd := exec.Command("sh", "-c", cmdString)
	resultBytes, err := cmd.Output()
	if err != nil {
		return LintResult{
			SQL:  sql,
			Ok:   false,
			Lint: string(resultBytes),
		}
	}

	return LintResult{
		SQL: sql,
		Ok:  true,
	}
}
