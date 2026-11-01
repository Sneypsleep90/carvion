package application

import (
	"context"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

// AdapterGateway is implemented by local process or future remote adapters.
type AdapterGateway interface {
	List(context.Context) ([]domain.AdapterInfo, error)
	GetInfo(context.Context, string) (domain.AdapterInfo, error)
	Detect(context.Context, string, domain.DetectRequest) (domain.DetectionResult, error)
	CheckCompatibility(context.Context, string, domain.CompatibilityRequest) (domain.CompatibilityResult, error)
	InspectResources(context.Context, string, domain.InspectResourcesRequest) (domain.ResourceInspection, error)
}

// CatalogRepository is the storage port used by application services. A local
// PostgreSQL implementation is wired during development; a Hub client can
// implement the same contract without changing application behavior.
type CatalogRepository interface {
	ListManufacturers(context.Context) ([]domain.Manufacturer, error)
	ListVehicleModels(context.Context, string) ([]domain.VehicleModel, error)
	ListInfotainmentPlatforms(context.Context, string) ([]domain.InfotainmentPlatform, error)
}
