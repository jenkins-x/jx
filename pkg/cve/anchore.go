package cve

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"fmt"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/table"
	"github.com/jenkins-x/jx/pkg/util"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	getVulnerabilitiesByImageID     = "/images/by_id/%s/vuln/%s"
	getVulnerabilitiesByImageDigest = "/images/%s"
	vulnerabilityType               = "os"
	getImages                       = "/images"
)

type result interface {
}

type VulnerabilityList struct {
	ImageDigest     string
	Vulnerabilities []Vulnerability
}

type Vulnerability struct {
	Fix      string
	Package  string
	Severity string
	URL      string
	Vuln     string
}

type Image struct {
	AnalysisStatus string        `json:"analysis_status,omitempty"`
	ImageDetails   []ImageDetail `json:"image_detail,omitempty"`
}

type ImageDetail struct {
	Registry string
	Repo     string
	Tag      string
	ImageId  string
	Fulltag  string
}

// AnchoreProvider implements CVEProvider interface for anchore.io
type AnchoreProvider struct {
	Client    *http.Client
	BasicAuth string
	BaseURL   string
}

func NewAnchoreProvider(server *auth.AuthServer, user *auth.UserAuth) (CVEProvider, error) {

	basicAuth := util.BasicAuth(user.Username, user.Password)

	provider := AnchoreProvider{
		BaseURL:   server.URL,
		BasicAuth: basicAuth,
		Client:    http.DefaultClient,
	}

	return &provider, nil
}

func (a AnchoreProvider) GetImageVulnerabilityTable(jxClient versioned.Interface, client kubernetes.Interface, table *table.Table, query CVEQuery) error {

	var err error
	var vList VulnerabilityList
	var imageIDs []string

	if query.ImageID != "" {
		var vList VulnerabilityList
		subPath := fmt.Sprintf(getVulnerabilitiesByImageID, query.ImageID, vulnerabilityType)

		err = a.anchoreGet(subPath, &vList)
		if err != nil {
			return fmt.Errorf("error getting vulnerabilities for image %s: %v", query.ImageID, err)
		}

		return a.addVulnerabilitiesTableRows(table, &vList)
	}

	if query.Environment != "" {

		var vList VulnerabilityList
		// list pods in the namespace
		podList, err := client.CoreV1().Pods(query.TargetNamespace).List(meta_v1.ListOptions{})
		if err != nil {
			return err
		}
		// if they have the annotation add the value to a list
		for _, p := range podList.Items {
			if p.Annotations[AnnotationCVEImageId] != "" {
				imageIDs = append(imageIDs, p.Annotations[AnnotationCVEImageId])
			}
		}
		// loop over the list and get the CVEs for each, adding the rows
		err = a.getCVEsFromImageList(table, &vList, imageIDs)
		if err != nil {
			return err
		}
	}

	// see if we can match images using an image name and optional version
	if query.ImageID == "" {
		// if we have an image name then lets try and match image id(s)
		if query.ImageName != "" {

			var images []Image
			subPath := fmt.Sprintf(getImages)

			err = a.anchoreGet(subPath, &images)
			if err != nil {
				return fmt.Errorf("error getting images %v", err)
			}

			for _, image := range images {
				for _, d := range image.ImageDetails {
					if d.Repo == query.ImageName {
						// if user has provided a version and it doesn't match lets skip this image
						if query.Vesion != "" && query.Vesion != d.Tag {
							continue
						}
						imageIDs = append(imageIDs, d.ImageId)
					}
				}
			}
			if len(imageIDs) > 0 {
				err = a.getCVEsFromImageList(table, &vList, imageIDs)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("no matching images found for ImageName %s and Vesion %s", query.ImageName, query.Vesion)
			}
		}
	} else {
		return fmt.Errorf("choose an image name, an optinal version or anchore image id to find vulnerabilities")
	}

	return nil

}

func (a AnchoreProvider) anchoreGet(subPath string, rs result) error {

	url := fmt.Sprintf("%s%s", a.BaseURL, subPath)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Basic "+a.BasicAuth)

	resp, err := a.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting vulnerabilities from anchore engine %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error response getting vulnerabilities from anchore engine: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(data, &rs)
	if err != nil {
		return fmt.Errorf("error unmarshalling %v", err)
	}
	return nil
}

func (a AnchoreProvider) addVulnerabilitiesTableRows(table *table.Table, vList *VulnerabilityList) error {

	var image []Image
	subPath := fmt.Sprintf(getVulnerabilitiesByImageDigest, vList.ImageDigest)

	err := a.anchoreGet(subPath, &image)
	if err != nil {
		return fmt.Errorf("error getting image for image digest %s: %v", vList.ImageDigest, err)
	}
	// TODO sort vList on severity and version?

	for _, v := range vList.Vulnerabilities {
		var sev string
		switch v.Severity {
		case "High":
			sev = util.ColorError(v.Severity)
		case "Medium":
			sev = util.ColorWarning(v.Severity)
		case "Low":
			sev = util.ColorStatus(v.Severity)
		}
		table.AddRow(image[0].ImageDetails[0].Fulltag, sev, v.Vuln, v.URL, v.Package, v.Fix)
	}
	return nil
}

func (a AnchoreProvider) getCVEsFromImageList(table *table.Table, vList *VulnerabilityList, ids []string) error {
	for _, imageID := range ids {
		subPath := fmt.Sprintf(getVulnerabilitiesByImageID, imageID, vulnerabilityType)

		err := a.anchoreGet(subPath, &vList)
		if err != nil {
			return fmt.Errorf("error getting vulnerabilities for image %s: %v", imageID, err)
		}

		err = a.addVulnerabilitiesTableRows(table, vList)
		if err != nil {
			return fmt.Errorf("error building vulnerabilities table for image digest %s: %v", vList.ImageDigest, err)
		}
	}
	return nil
}
