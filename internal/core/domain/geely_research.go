package domain

type GeelyResearchEvidence struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Publisher   string `json:"publisher"`
	URL         string `json:"url"`
	PublishedAt string `json:"publishedAt,omitempty"`
	RetrievedAt string `json:"retrievedAt"`
	Scope       string `json:"scope"`
	Confidence  string `json:"confidence"`
}

// GeelyResearchProfile contains file-reported values and their provenance.
// Imported profile data is never equivalent to live vehicle detection.
type GeelyResearchProfile struct {
	Schema             string `json:"schema"`
	ProfileID          string `json:"profileId"`
	Manufacturer       string `json:"manufacturer,omitempty"`
	Model              string `json:"model,omitempty"`
	ModelYear          int    `json:"modelYear,omitempty"`
	Market             string `json:"market,omitempty"`
	Trim               string `json:"trim,omitempty"`
	Platform           string `json:"platform,omitempty"`
	OSName             string `json:"osName,omitempty"`
	OSVersion          string `json:"osVersion,omitempty"`
	HardwareIdentifier string `json:"hardwareIdentifier,omitempty"`
	HardwareRevision   string `json:"hardwareRevision,omitempty"`
	FirmwareVersion    string `json:"firmwareVersion,omitempty"`
	FirmwareBuildID    string `json:"firmwareBuildId,omitempty"`
	Origin             string `json:"origin"`
}

type GeelyResearchAnalysis struct {
	Profile       GeelyResearchProfile `json:"profile"`
	Detection     DetectionResult      `json:"detection"`
	Compatibility CompatibilityResult  `json:"compatibility"`
}

type GeelyResourceRequest struct {
	Filename string             `json:"filename"`
	Format   LocalizationFormat `json:"format"`
	Content  string             `json:"content"`
}

type GeelyLocalizationImportRequest struct {
	ProfileJSON string             `json:"profileJson"`
	Name        string             `json:"name"`
	Locale      string             `json:"locale"`
	Filename    string             `json:"filename"`
	Format      LocalizationFormat `json:"format"`
	Content     string             `json:"content"`
}

type GeelyLocalizationImportResult struct {
	Project    LocalizationProject   `json:"project"`
	Analysis   GeelyResearchAnalysis `json:"analysis"`
	Inspection ResourceInspection    `json:"inspection"`
}

type GeelyDiagnosticReportRequest struct {
	ProfileJSON string                `json:"profileJson"`
	Resource    *GeelyResourceRequest `json:"resource,omitempty"`
}

type GeelyDiagnosticReport struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}
