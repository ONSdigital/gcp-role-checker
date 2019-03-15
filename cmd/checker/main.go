package main

import "C"

import (
	"flag"
	"path"
	"runtime"
	"strings"

	"github.com/ONSdigital/gcp-role-checker/checker"
)

func main() {
	organizationPtr := flag.String(
		"org", "",
		"The organization resource ID I.e. organizations/999999999999")

	labelsPtr := flag.String(
		"project_labels", "",
		"A set of labels to filter projects on \n"+
			"I.e. env:dev,project:foo")

	dataDirPtr := flag.String(
		"data", "",
		"Directory where data should be output to",
	)

	flag.Parse()
	labels := parseLabels(*labelsPtr)
	dataDir := getDataDir(*dataDirPtr)

	checker.RunChecker(*organizationPtr, labels, dataDir)
}

func parseLabels(rawLabels string) map[string]string {
	labels := make(map[string]string)
	for _, rawLabelMap := range strings.Split(rawLabels, ",") {
		if len(rawLabelMap) > 0 {
			labelMap := strings.Split(rawLabelMap, ":")
			labels[labelMap[0]] = labelMap[1]
		}
	}

	return labels
}

func getDataDir(rawDataDir string) string {
	if len(rawDataDir) > 0 {
		return rawDataDir
	}

	_, filename, _, _ := runtime.Caller(0)
	packageRoot := path.Dir(path.Dir(path.Dir(filename)))
	return path.Join(packageRoot, "data")
}
