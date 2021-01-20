package version

// AppVersion structure for version.
type AppVersion struct {
	Version   string
	GitCommit string
	BuildDate string
}

var (
	// Version is the current version.
	Version = ""
	// Metadata is an extra.
	Metadata = "unreleased"
	// GitCommit is a git sha1.
	GitCommit = ""
	// BuildDate is the build date.
	BuildDate = ""
)

func buildVersion() string {
	// Check if metadata are not present
	if Metadata == "" {
		return Version
	}

	return Version + "-" + Metadata
}

// GetVersion is here to get version of the cli.
func GetVersion() *AppVersion {
	return &AppVersion{
		Version:   buildVersion(),
		GitCommit: GitCommit,
		BuildDate: BuildDate,
	}
}
