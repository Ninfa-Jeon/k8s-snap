package helm

// InstallableChart describes a chart that can be deployed on a running cluster.
type InstallableChart struct {
	// Name is the name of the chart.
	Name string

	// Version is the version of the chart.
	Version string

	// InstallName is the install name of the chart.
	InstallName string

	// InstallNamespace is the namespace to install the chart.
	InstallNamespace string
}
