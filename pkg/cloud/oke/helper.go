package oke

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os/exec"
	"sort"
	"strings"
)

func GetOptionValues() ([]string, []string, []string, string, error) {
	jsonString, err := exec.Command("oci", "ce", "node-pool-options", "get", "--node-pool-option-id", "all").Output()
	if err != nil {
		return nil, nil, nil, "", err
	}
	var dat map[string]interface{}
	if err := json.Unmarshal(jsonString, &dat); err != nil {
		logrus.Info("error")
		return nil, nil, nil, "", err
	}

	originalStrs := dat["data"].(map[string]interface{})

	kubeVersions := fmt.Sprintf("%v", originalStrs["kubernetes-versions"])
	kubeVersions = strings.TrimPrefix(kubeVersions, "[")
	kubeVersions = strings.TrimSuffix(kubeVersions, "]")
	kubeVersionsArray := strings.Split(kubeVersions, " ")

	images := fmt.Sprintf("%v", originalStrs["images"])
	images = strings.TrimPrefix(images, "[")
	images = strings.TrimSuffix(images, "]")
	imagesArray := strings.Split(images, " ")

	shapes := fmt.Sprintf("%v", originalStrs["shapes"])
	shapes = strings.TrimPrefix(shapes, "[")
	shapes = strings.TrimSuffix(shapes, "]")
	shapesArray := strings.Split(shapes, " ")

	sort.Strings(kubeVersionsArray)

	return imagesArray, kubeVersionsArray, shapesArray, kubeVersionsArray[0], nil
}
