package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: pool}
}

func (r *CatalogRepository) ListManufacturers(ctx context.Context) ([]domain.Manufacturer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name
		FROM manufacturers
		ORDER BY name, id`)
	if err != nil {
		return nil, catalogError(ctx, "list manufacturers", err)
	}
	defer rows.Close()

	items := make([]domain.Manufacturer, 0)
	for rows.Next() {
		var item domain.Manufacturer
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, catalogError(ctx, "scan manufacturer", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, catalogError(ctx, "read manufacturers", err)
	}
	return items, nil
}

func (r *CatalogRepository) ListVehicleModels(ctx context.Context, manufacturerID string) ([]domain.VehicleModel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT vm.id::text, vm.manufacturer_id::text, m.name, vm.name
		FROM vehicle_models AS vm
		JOIN manufacturers AS m ON m.id = vm.manufacturer_id
		WHERE ($1::uuid IS NULL OR vm.manufacturer_id = $1::uuid)
		ORDER BY m.name, vm.name, vm.id`, nullableID(manufacturerID))
	if err != nil {
		return nil, catalogError(ctx, "list vehicle models", err)
	}
	defer rows.Close()

	items := make([]domain.VehicleModel, 0)
	for rows.Next() {
		var item domain.VehicleModel
		if err := rows.Scan(&item.ID, &item.ManufacturerID, &item.ManufacturerName, &item.Name); err != nil {
			return nil, catalogError(ctx, "scan vehicle model", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, catalogError(ctx, "read vehicle models", err)
	}
	return items, nil
}

func (r *CatalogRepository) ListInfotainmentPlatforms(ctx context.Context, manufacturerID string) ([]domain.InfotainmentPlatform, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text, p.name, p.manufacturer_id::text, m.name
		FROM infotainment_platforms AS p
		LEFT JOIN manufacturers AS m ON m.id = p.manufacturer_id
		WHERE ($1::uuid IS NULL OR p.manufacturer_id = $1::uuid)
	ORDER BY p.name, p.id`, nullableID(manufacturerID))
	if err != nil {
		return nil, catalogError(ctx, "list infotainment platforms", err)
	}
	defer rows.Close()

	items := make([]domain.InfotainmentPlatform, 0)
	for rows.Next() {
		var item domain.InfotainmentPlatform
		var manufacturerID, manufacturerName pgtype.Text
		if err := rows.Scan(&item.ID, &item.Name, &manufacturerID, &manufacturerName); err != nil {
			return nil, catalogError(ctx, "scan infotainment platform", err)
		}
		if manufacturerID.Valid {
			item.ManufacturerID = &manufacturerID.String
		}
		if manufacturerName.Valid {
			item.ManufacturerName = &manufacturerName.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, catalogError(ctx, "read infotainment platforms", err)
	}
	return items, nil
}

func nullableID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func catalogError(ctx context.Context, operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: %s: %v", domain.ErrCatalogUnavailable, operation, err)
}
