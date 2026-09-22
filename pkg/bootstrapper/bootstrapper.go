package bootstrapper

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.goms.io/aks/AKSFlexNode/pkg/config"
)

// Bootstrapper executes bootstrap steps sequentially
type Bootstrapper struct {
	*BaseExecutor
}

// New creates a new bootstrapper
func New(cfg *config.Config, logger *logrus.Logger) *Bootstrapper {
	return &Bootstrapper{
		BaseExecutor: NewBaseExecutor(cfg, logger),
	}
}

// Bootstrap executes all bootstrap steps sequentially
func (b *Bootstrapper) Bootstrap(ctx context.Context) (*ExecutionResult, error) {
	steps := b.getBootstrapSteps()
	return b.ExecuteSteps(ctx, steps, "bootstrap")
}

// Unbootstrap executes all cleanup steps sequentially (in reverse order of bootstrap)
func (b *Bootstrapper) Unbootstrap(ctx context.Context) (*ExecutionResult, error) {
	steps := b.getUnbootstrapSteps()
	return b.ExecuteSteps(ctx, steps, "unbootstrap")
}
