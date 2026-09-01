package buildenv

// CacheDir is a non-secret build-time value. A validated Build Profile may
// replace it with -ldflags -X. It is deliberately separate from the
// environment-variable names below: there is no runtime environment override.
var CacheDir = "~/.mediakit"

// Environment-variable names are variables so a validated Build Profile can
// replace the names at link time with -ldflags -X. Values are always read at
// runtime and must never be embedded in the binary.
var (
	CloudAPIKey           = "MEDIAKIT_API_KEY"
	CloudLogID            = "MEDIAKIT_LOGID"
	CloudEndpoint         = "MEDIAKIT_ENDPOINT"
	Surface               = "MEDIAKIT_SURFACE"
	TaskSource            = "MEDIAKIT_TASK_SOURCE"
	Runtime               = "MEDIAKIT_RUNTIME"
	OutputPath            = "MEDIAKIT_OUTPUT_PATH"
	DisableUpdateCheck    = "MEDIAKIT_DISABLE_UPDATE_CHECK"
	UpdateInternalRefresh = "MEDIAKIT_INTERNAL_REFRESH"
)
