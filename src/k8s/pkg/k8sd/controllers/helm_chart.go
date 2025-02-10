package controllers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/canonical/k8s/pkg/k8sd/charts"
	"github.com/canonical/k8s/pkg/k8sd/database"
	"github.com/canonical/k8s/pkg/log"
	"github.com/canonical/k8s/pkg/snap"
	"github.com/canonical/microcluster/v2/state"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var ErrChartExists = errors.New("helm chart already exists")
var ErrChartParseFailed = errors.New("failed to parse helm chart")

// HelmChartController is a controller that syncs helm charts available in the snap with the microcluster database.
type HelmChartController struct {
	snap      snap.Snap
	waitReady func()
}

// NewHelmChartController creates a new helm chart controller.
func NewHelmChartController(snap snap.Snap, waitReady func()) *HelmChartController {
	return &HelmChartController{
		snap:      snap,
		waitReady: waitReady,
	}
}

// Run runs the helm chart controller.
// The controller reconciles helm charts available in the snap with the microcluster database until all valid charts are inserted.
func (c *HelmChartController) Run(ctx context.Context, s state.State) {
	ctx = log.NewContext(ctx, log.FromContext(ctx).WithValues("controller", "helm-chart"))
	log := log.FromContext(ctx)

	log.Info("Waiting for node to be ready")
	// wait for microcluster node to be ready
	c.waitReady()

	log.Info("Starting helm chart controller")

	for {
		log.Info("Reconciling helm charts")

		var retryRequired bool

		for _, chartFS := range charts.Charts() {
			entries, err := chartFS.ReadDir("charts")
			if err != nil {
				log.Error(err, "failed to read helm charts directory")
				retryRequired = true
				continue
			}

			for _, e := range entries {
				if e.IsDir() {
					// TODO(berkayoz): Add support for directories by creating a tarball(.tgz) automatically
					log.WithValues("entry", e.Name()).Info("skipping reconciliation of directory")
					continue
				}

				chartBytes, err := chartFS.ReadFile(filepath.Join("charts", e.Name()))
				if err != nil {
					log.WithValues("entry", e.Name()).Error(err, "failed to read helm chart")
					retryRequired = true
					continue
				}

				if err := c.reconcile(ctx, s, chartBytes); err != nil {
					if errors.Is(err, ErrChartExists) || errors.Is(err, ErrChartParseFailed) {
						log.WithValues("entry", e.Name()).Info("skipping reconciliation of entry")
					} else {
						log.WithValues("entry", e.Name()).Error(err, "failed to reconcile helm chart")
						retryRequired = true
					}
				}
			}
		}

		if !retryRequired {
			log.Info("Reconcilation of helm charts complete")
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}
}

func (c *HelmChartController) reconcile(ctx context.Context, s state.State, chartBytes []byte) error {
	log := log.FromContext(ctx)

	chart, err := loader.LoadArchive(bytes.NewReader(chartBytes))
	if err != nil {
		return errors.Join(ErrChartParseFailed, err)
	}

	var exists bool
	if err := s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		exists, err = database.HelmChartExists(ctx, tx, chart.Metadata.Name, chart.Metadata.Version)
		return err
	}); err != nil {
		return fmt.Errorf("failed to check if helm chart exists: %w", err)
	}

	if exists {
		return ErrChartExists
	}

	if err := s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		err := database.InsertHelmChart(ctx, tx, chart.Metadata.Name, chart.Metadata.Version, chartBytes)
		return err
	}); err != nil {
		return fmt.Errorf("failed to insert helm chart: %w", err)
	}

	log.WithValues("chart", chart.Metadata.Name, "version", chart.Metadata.Version).Info("helm chart inserted successfully")

	return nil
}
