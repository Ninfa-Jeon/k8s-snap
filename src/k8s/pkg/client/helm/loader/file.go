package loader

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/canonical/k8s/pkg/client/helm"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// fileLoader is a helm chart loader that loads charts from the filesystem.
type fileLoader struct {
	manifestsBaseDir string
}

// NewFileLoader creates a new file loader.
func NewFileLoader(manifestsBaseDir string) *fileLoader {
	return &fileLoader{
		manifestsBaseDir: manifestsBaseDir,
	}
}

// Load loads a helm chart from the filesystem by name and version.
func (l *fileLoader) Load(ctx context.Context, f helm.InstallableChart) (*chart.Chart, error) {
	chart, err := loader.Load(filepath.Join(l.manifestsBaseDir, fmt.Sprintf("%s-%s.tgz", f.Name, f.Version)))
	if err != nil {
		return nil, fmt.Errorf("failed to load helm chart: %w", err)
	}

	return chart, nil
}
