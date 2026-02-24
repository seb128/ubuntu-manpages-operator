package transform

// ManpageMeta holds package metadata associated with a manpage file.
type ManpageMeta struct {
	PackageName    string
	PackageVersion string
	SourcePackage  string
	Filename       string // base filename (e.g. "ls.1.gz"), used as title fallback
}
