package quickstarts

type Quickstart struct {
	ID             string
	Owner          string
	Name           string
	Language       string
	Framework      string
	Tags           []string
	DownloadZipURL string
}

type QuickstartModel struct {
	Quickstarts map[string]*Quickstart
}

func NewQuickstartModel() *QuickstartModel {
	return &QuickstartModel{
		Quickstarts: map[string]*Quickstart{},
	}
}

type QuickstartFilter struct {
	Language  string
	Framework string
	Owner     string
	Text      string
	Tags      []string
}
