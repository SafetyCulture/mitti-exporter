package feed

// ActionField represents a row from the action_fields feed
type ActionField struct {
	ActionID   string `json:"action_id" csv:"action_id" gorm:"primarykey;size:64"`
	FieldID    string `json:"field_id" csv:"field_id" gorm:"primarykey;size:128"`
	FieldType  string `json:"field_type" csv:"field_type" gorm:"size:50"`
	FieldValue string `json:"field_value" csv:"field_value"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (ActionField) TableName() string {
	return "action_fields"
}

// NewActionFieldFeed returns the feed for the custom field values of actions
func NewActionFieldFeed(limit int) Feed {
	return &rowFeed[ActionField]{
		name:         "action_fields",
		initialURL:   "/tasks/v1/feed/action_fields",
		primaryKey:   []string{"action_id", "field_id"},
		columns:      []string{"field_type", "field_value"},
		order:        "action_id, field_id",
		hasRemaining: true,
		params:       GetFeedParams{Limit: limit},
		key:          func(r *ActionField) string { return r.ActionID + "|" + r.FieldID },
	}
}
