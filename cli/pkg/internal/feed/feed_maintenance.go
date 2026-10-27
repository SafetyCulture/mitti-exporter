package feed

// MaintenanceProgram represents a row from the maintenance_programs feed
type MaintenanceProgram struct {
	ID          string `json:"id" csv:"id" gorm:"primarykey;size:128"`
	Name        string `json:"name" csv:"name"`
	Description string `json:"description" csv:"description"`
	AssetsCount int32  `json:"assets_count" csv:"assets_count"`
	PlansCount  int32  `json:"plans_count" csv:"plans_count"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (MaintenanceProgram) TableName() string {
	return "maintenance_programs"
}

// MaintenancePlan represents a row from the maintenance_plans feed
type MaintenancePlan struct {
	ID            string `json:"id" csv:"id" gorm:"primarykey;size:128"`
	ProgramID     string `json:"program_id" csv:"program_id" gorm:"size:128"`
	Name          string `json:"name" csv:"name"`
	Description   string `json:"description" csv:"description"`
	IntervalType  string `json:"interval_type" csv:"interval_type" gorm:"size:50"`
	IntervalValue string `json:"interval_value" csv:"interval_value" gorm:"size:50"`
	IntervalUnit  string `json:"interval_unit" csv:"interval_unit" gorm:"size:50"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (MaintenancePlan) TableName() string {
	return "maintenance_plans"
}

// AssetMaintenance represents a row from the assets_maintenance feed.
// It holds the maintenance status of an asset for each plan the asset is subject to
type AssetMaintenance struct {
	AssetID                 string `json:"asset_id" csv:"asset_id" gorm:"primarykey;size:128"`
	PlanID                  string `json:"plan_id" csv:"plan_id" gorm:"primarykey;size:128"`
	AssetCode               string `json:"asset_code" csv:"asset_code"`
	ProgramID               string `json:"program_id" csv:"program_id" gorm:"size:128"`
	ProgramName             string `json:"program_name" csv:"program_name"`
	PlanName                string `json:"plan_name" csv:"plan_name"`
	IntervalType            string `json:"interval_type" csv:"interval_type" gorm:"size:50"`
	ServiceStatus           string `json:"service_status" csv:"service_status" gorm:"size:50"`
	LastServiceValue        string `json:"last_service_value" csv:"last_service_value"`
	NextServiceValue        string `json:"next_service_value" csv:"next_service_value"`
	CurrentReadingValue     string `json:"current_reading_value" csv:"current_reading_value"`
	DueInValue              string `json:"due_in_value" csv:"due_in_value"`
	LastServiceTimestamp    string `json:"last_service_timestamp" csv:"last_service_timestamp" gorm:"size:40"`
	CurrentReadingTimestamp string `json:"current_reading_timestamp" csv:"current_reading_timestamp" gorm:"size:40"`
	CurrentReadingSource    string `json:"current_reading_source" csv:"current_reading_source"`
	IntervalValue           string `json:"interval_value" csv:"interval_value" gorm:"size:50"`
	IntervalUnit            string `json:"interval_unit" csv:"interval_unit" gorm:"size:50"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (AssetMaintenance) TableName() string {
	return "assets_maintenance"
}

// NewMaintenanceProgramFeed returns the feed for /assets/v1/feed/maintenance_programs
func NewMaintenanceProgramFeed(limit int) Feed {
	return &rowFeed[MaintenanceProgram]{
		name:       "maintenance_programs",
		initialURL: "/assets/v1/feed/maintenance_programs",
		primaryKey: []string{"id"},
		columns:    []string{"name", "description", "assets_count", "plans_count"},
		order:      "id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *MaintenanceProgram) string { return r.ID },
	}
}

// NewMaintenancePlanFeed returns the feed for /assets/v1/feed/maintenance_plans
func NewMaintenancePlanFeed(limit int) Feed {
	return &rowFeed[MaintenancePlan]{
		name:       "maintenance_plans",
		initialURL: "/assets/v1/feed/maintenance_plans",
		primaryKey: []string{"id"},
		columns:    []string{"program_id", "name", "description", "interval_type", "interval_value", "interval_unit"},
		order:      "program_id, id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *MaintenancePlan) string { return r.ID },
	}
}

// NewAssetMaintenanceFeed returns the feed for /assets/v1/feed/assets_maintenance
func NewAssetMaintenanceFeed(limit int) Feed {
	return &rowFeed[AssetMaintenance]{
		name:       "assets_maintenance",
		initialURL: "/assets/v1/feed/assets_maintenance",
		primaryKey: []string{"asset_id", "plan_id"},
		order:      "asset_id, plan_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *AssetMaintenance) string { return r.AssetID + "|" + r.PlanID },
		columns: []string{
			"asset_code", "program_id", "program_name", "plan_name", "interval_type", "service_status",
			"last_service_value", "next_service_value", "current_reading_value", "due_in_value",
			"last_service_timestamp", "current_reading_timestamp", "current_reading_source", "interval_value", "interval_unit",
		},
	}
}
