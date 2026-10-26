package feed

import "time"

// ScheduleAssigneeV2 represents a row from the Scheduling feed service schedule_assignees feed.
// The legacy schedule_assignees columns come first, then the target
type ScheduleAssigneeV2 struct {
	ID             string    `json:"id" csv:"id" gorm:"primarykey;size:375"`
	ScheduleID     string    `json:"schedule_id" csv:"schedule_id" gorm:"size:128"`
	AssigneeID     string    `json:"assignee_id" csv:"assignee_id" gorm:"size:256"`
	OrganisationID string    `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	Type           string    `json:"type" csv:"type" gorm:"size:30"`
	Name           string    `json:"name" csv:"name"`
	ExportedAt     time.Time `json:"-" csv:"exported_at" gorm:"autoUpdateTime"`
	// TargetID is the site or asset UUID of the row, empty for schedules without sites or assets.
	// It is part of the primary key, as the API returns the same id once per site or asset
	TargetID   string `json:"target_id" csv:"target_id" gorm:"primarykey;size:36"`
	TargetType string `json:"target_type" csv:"target_type" gorm:"size:10"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (ScheduleAssigneeV2) TableName() string {
	return "new_schedule_assignees"
}

// NewScheduleAssigneeV2Feed returns the feed for /scheduling/v1/feed/schedule_assignees.
// statuses filters on the schedule status (ACTIVE, PAUSED, FINISHED); it is empty for every status
func NewScheduleAssigneeV2Feed(templateIDs []string, statuses []string) Feed {
	return &rowFeed[ScheduleAssigneeV2]{
		name:       "new_schedule_assignees",
		initialURL: withScheduleStatuses("/scheduling/v1/feed/schedule_assignees", statuses),
		primaryKey: []string{"id", "target_id"},
		order:      "schedule_id, assignee_id, target_id",
		params: GetFeedParams{
			TemplateIDs: templateIDs,
			Limit:       maxSchedulingFeedLimit,
		},
		key:     func(r *ScheduleAssigneeV2) string { return r.ID + "|" + r.TargetID },
		columns: []string{"schedule_id", "assignee_id", "organisation_id", "type", "name", "exported_at", "target_type"},
	}
}
