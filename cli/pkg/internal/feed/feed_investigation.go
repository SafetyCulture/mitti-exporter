package feed

// Investigation represents a row from the investigations feed
type Investigation struct {
	InvestigationID         string `json:"investigation_id" csv:"investigation_id" gorm:"primarykey;size:128"`
	Title                   string `json:"title" csv:"title"`
	Description             string `json:"description" csv:"description"`
	CreatedAt               string `json:"created_at" csv:"created_at" gorm:"size:40"`
	ModifiedAt              string `json:"modified_at" csv:"modified_at" gorm:"size:40"`
	StatusID                string `json:"status_id" csv:"status_id" gorm:"size:128"`
	StatusTitle             string `json:"status_title" csv:"status_title"`
	StatusBadgeColor        int32  `json:"status_badge_color" csv:"status_badge_color"`
	IdentifierPrefix        string `json:"identifier_prefix" csv:"identifier_prefix" gorm:"size:50"`
	IdentifierSequence      int32  `json:"identifier_sequence" csv:"identifier_sequence"`
	IdentifierForDisplay    string `json:"identifier_for_display" csv:"identifier_for_display"`
	CreatorID               string `json:"creator_id" csv:"creator_id" gorm:"size:128"`
	OwnerID                 string `json:"owner_id" csv:"owner_id" gorm:"size:128"`
	ModifiedByID            string `json:"modified_by_id" csv:"modified_by_id" gorm:"size:128"`
	CategoryID              string `json:"category_id" csv:"category_id" gorm:"size:128"`
	CategoryTitle           string `json:"category_title" csv:"category_title"`
	CategoryDescription     string `json:"category_description" csv:"category_description"`
	CategoryTitleAtCreation string `json:"category_title_at_creation" csv:"category_title_at_creation"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (Investigation) TableName() string {
	return "investigations"
}

// InvestigationField represents a row from the investigation_fields feed
type InvestigationField struct {
	InvestigationID string `json:"investigation_id" csv:"investigation_id" gorm:"primarykey;size:128"`
	FieldInstanceID string `json:"field_instance_id" csv:"field_instance_id" gorm:"primarykey;size:128"`
	FieldID         string `json:"field_id" csv:"field_id" gorm:"size:128"`
	Title           string `json:"title" csv:"title"`
	Content         string `json:"content" csv:"content"`
	CreatedAt       string `json:"created_at" csv:"created_at" gorm:"size:40"`
	ModifiedAt      string `json:"modified_at" csv:"modified_at" gorm:"size:40"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (InvestigationField) TableName() string {
	return "investigation_fields"
}

// InvestigationDetailField represents a row from the investigation_detail_fields feed.
// The multi select, multi site and select options columns contain JSON arrays
type InvestigationDetailField struct {
	InvestigationID       string `json:"investigation_id" csv:"investigation_id" gorm:"primarykey;size:128"`
	DetailFieldInstanceID string `json:"detail_field_instance_id" csv:"detail_field_instance_id" gorm:"primarykey;size:128"`
	DetailFieldID         string `json:"detail_field_id" csv:"detail_field_id" gorm:"size:128"`
	DetailFieldType       string `json:"detail_field_type" csv:"detail_field_type" gorm:"size:50"`
	Title                 string `json:"title" csv:"title"`
	SingleSelectContent   string `json:"single_select_content" csv:"single_select_content"`
	MultiSelectContent    string `json:"multi_select_content" csv:"multi_select_content"`
	DateTimeContent       string `json:"date_time_content" csv:"date_time_content"`
	SiteContent           string `json:"site_content" csv:"site_content"`
	MultiSiteContent      string `json:"multi_site_content" csv:"multi_site_content"`
	SelectOptions         string `json:"select_options" csv:"select_options"`
	CreatedAt             string `json:"created_at" csv:"created_at" gorm:"size:40"`
	ModifiedAt            string `json:"modified_at" csv:"modified_at" gorm:"size:40"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (InvestigationDetailField) TableName() string {
	return "investigation_detail_fields"
}

// InvestigationRelationship represents a row from the investigation_relationships feed.
// It links an investigation to another resource such as an issue, action or inspection
type InvestigationRelationship struct {
	InvestigationID string `json:"investigation_id" csv:"investigation_id" gorm:"primarykey;size:128"`
	ResourceType    string `json:"resource_type" csv:"resource_type" gorm:"primarykey;size:50"`
	ResourceID      string `json:"resource_id" csv:"resource_id" gorm:"primarykey;size:128"`
	LinkedAt        string `json:"linked_at" csv:"linked_at" gorm:"size:40"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (InvestigationRelationship) TableName() string {
	return "investigation_relationships"
}

// NewInvestigationFeed returns the feed for /incidents/v1/feed/investigations
func NewInvestigationFeed(limit int) Feed {
	return &rowFeed[Investigation]{
		name:       "investigations",
		initialURL: "/incidents/v1/feed/investigations",
		primaryKey: []string{"investigation_id"},
		order:      "investigation_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *Investigation) string { return r.InvestigationID },
		columns: []string{
			"title", "description", "created_at", "modified_at", "status_id", "status_title", "status_badge_color",
			"identifier_prefix", "identifier_sequence", "identifier_for_display", "creator_id", "owner_id",
			"modified_by_id", "category_id", "category_title", "category_description", "category_title_at_creation",
		},
	}
}

// NewInvestigationFieldFeed returns the feed for /incidents/v1/feed/investigation_fields
func NewInvestigationFieldFeed(limit int) Feed {
	return &rowFeed[InvestigationField]{
		name:       "investigation_fields",
		initialURL: "/incidents/v1/feed/investigation_fields",
		primaryKey: []string{"investigation_id", "field_instance_id"},
		columns:    []string{"field_id", "title", "content", "created_at", "modified_at"},
		order:      "investigation_id, field_instance_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *InvestigationField) string { return r.InvestigationID + "|" + r.FieldInstanceID },
	}
}

// NewInvestigationDetailFieldFeed returns the feed for /incidents/v1/feed/investigation_detail_fields
func NewInvestigationDetailFieldFeed(limit int) Feed {
	return &rowFeed[InvestigationDetailField]{
		name:       "investigation_detail_fields",
		initialURL: "/incidents/v1/feed/investigation_detail_fields",
		primaryKey: []string{"investigation_id", "detail_field_instance_id"},
		order:      "investigation_id, detail_field_instance_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *InvestigationDetailField) string { return r.InvestigationID + "|" + r.DetailFieldInstanceID },
		columns: []string{
			"detail_field_id", "detail_field_type", "title", "single_select_content", "multi_select_content",
			"date_time_content", "site_content", "multi_site_content", "select_options", "created_at", "modified_at",
		},
	}
}

// NewInvestigationRelationshipFeed returns the feed for /incidents/v1/feed/investigation_relationships
func NewInvestigationRelationshipFeed(limit int) Feed {
	return &rowFeed[InvestigationRelationship]{
		name:       "investigation_relationships",
		initialURL: "/incidents/v1/feed/investigation_relationships",
		primaryKey: []string{"investigation_id", "resource_type", "resource_id"},
		columns:    []string{"linked_at"},
		order:      "investigation_id, resource_type, resource_id",
		params:     GetFeedParams{Limit: limit},
		key: func(r *InvestigationRelationship) string {
			return r.InvestigationID + "|" + r.ResourceType + "|" + r.ResourceID
		},
	}
}
