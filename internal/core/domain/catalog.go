package domain

type Manufacturer struct {
	ID   string
	Name string
}

type VehicleModel struct {
	ID               string
	ManufacturerID   string
	ManufacturerName string
	Name             string
}

type InfotainmentPlatform struct {
	ID               string
	Name             string
	ManufacturerID   *string
	ManufacturerName *string
}
