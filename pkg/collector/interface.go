package collector

type Collector interface {

	// CollectFiles collects the given file paths and collects them into the storage
	// relative to the given output path. Returns the list of URLs to access the files
	CollectFiles(patterns []string, outputPath string, basedir string) ([]string, error)
}
