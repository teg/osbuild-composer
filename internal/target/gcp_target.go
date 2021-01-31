package target

type GCPTargetOptions struct {
	Filename string `json:"filename"`
	Bucket   string `json:"bucket"`
	Object   string `json:"object"`
}

func (GCPTargetOptions) isTargetOptions() {}

func NewGCPTarget(options *GCPTargetOptions) *Target {
	return newTarget("org.osbuild.gcp", options)
}
