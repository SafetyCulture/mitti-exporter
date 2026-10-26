package feed

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ScheduleOccurrenceV2 represents a row from the Scheduling feed service schedule_occurrences feed.
// The legacy schedule_occurrences columns come first, then the columns the PRD adds
type ScheduleOccurrenceV2 struct {
	ID               string     `json:"id" csv:"id" gorm:"primarykey;size:375"`
	ScheduleID       string     `json:"schedule_id" csv:"schedule_id" gorm:"size:128"`
	OccurrenceID     string     `json:"occurrence_id" csv:"occurrence_id" gorm:"size:128"`
	TemplateID       string     `json:"template_id" csv:"template_id" gorm:"size:100"`
	OrganisationID   string     `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	StartTime        *time.Time `json:"start_time" csv:"start_time"`
	DueTime          *time.Time `json:"due_time" csv:"due_time"`
	MissTime         *time.Time `json:"miss_time" csv:"miss_time"`
	OccurrenceStatus string     `json:"occurrence_status" csv:"occurrence_status" gorm:"size:20"`
	AuditID          string     `json:"audit_id" csv:"audit_id" gorm:"size:100"`
	CompletedAt      *time.Time `json:"completed_at" csv:"completed_at"`
	ExportedAt       time.Time  `json:"-" csv:"exported_at" gorm:"autoUpdateTime"`
	UserID           string     `json:"assignee_id" csv:"user_id" gorm:"size:256"`
	AssigneeStatus   string     `json:"assignee_status" csv:"assignee_status" gorm:"size:20"`
	Note             string     `json:"-" csv:"note"`

	// AssigneeType is how the user was assigned: user, group, site, or site_intersection for a group within a site
	AssigneeType        string `json:"-" csv:"assignee_type" gorm:"size:20"`
	AssigneeUserID      string `json:"-" csv:"assignee_user_id" gorm:"size:37"`
	AssigneeUserName    string `json:"-" csv:"assignee_user_name"`
	AssigneeSiteID      string `json:"-" csv:"assignee_site_id" gorm:"size:41"`
	AssigneeSiteName    string `json:"-" csv:"assignee_site_name"`
	AssigneeGroupID     string `json:"-" csv:"assignee_group_id" gorm:"size:37"`
	AssigneeGroupName   string `json:"-" csv:"assignee_group_name"`
	CompletedByUserID   string `json:"completed_by_user_id" csv:"completed_by_user_id" gorm:"size:37"`
	CompletedByUserName string `json:"completed_by_user_name" csv:"completed_by_user_name"`
	// TargetID is the site or asset UUID of the row, empty for schedules without sites or assets.
	// It is part of the primary key, as the API returns the same id once per site or asset
	TargetID   string `json:"target_id" csv:"target_id" gorm:"primarykey;size:36"`
	TargetType string `json:"target_type" csv:"target_type" gorm:"size:10"`
	// TargetName is the name of the site, or the code of the asset
	TargetName string `json:"target_name" csv:"target_name"`
	WontDoNote string `json:"wont_do_note" csv:"wont_do_note"`
}

// UnmarshalJSON decodes the API fields and fills the assignment and legacy columns
func (o *ScheduleOccurrenceV2) UnmarshalJSON(b []byte) error {
	type apiOccurrence ScheduleOccurrenceV2
	if err := json.Unmarshal(b, (*apiOccurrence)(o)); err != nil {
		return fmt.Errorf("ScheduleOccurrenceV2.UnmarshalJSON: decode occurrence fields: %w", err)
	}

	var api struct {
		AssigneeFrom      string   `json:"assignee_from"`
		AssigneeFromNames []string `json:"assignee_from_names"`
	}
	if err := json.Unmarshal(b, &api); err != nil {
		return fmt.Errorf("ScheduleOccurrenceV2.UnmarshalJSON: decode assignee fields: %w", err)
	}

	o.Note = o.WontDoNote

	getName := func(i int) string {
		if i < len(api.AssigneeFromNames) {
			return api.AssigneeFromNames[i]
		}
		return ""
	}

	// assignee_from is user_<id>, role_<id>, location_<id> or location_<id>+role_<id> for a group within a site
	switch from := api.AssigneeFrom; {
	case strings.HasPrefix(from, "user_"):
		o.AssigneeType = "user"
		o.AssigneeUserID = from
		o.AssigneeUserName = getName(0)
	case strings.HasPrefix(from, "role_"):
		o.AssigneeType = "group"
		o.AssigneeGroupID = from
		o.AssigneeGroupName = getName(0)
	case strings.HasPrefix(from, "location_"):
		site, group, groupFound := strings.Cut(from, "+")
		o.AssigneeType = "site"
		o.AssigneeSiteID = site
		o.AssigneeSiteName = getName(0)
		if groupFound {
			o.AssigneeType = "site_intersection"
			o.AssigneeGroupID = group
			o.AssigneeGroupName = getName(1)
		}
	}
	return nil
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (ScheduleOccurrenceV2) TableName() string {
	return "new_schedule_occurrences"
}

// NewScheduleOccurrenceV2Feed returns the feed for /scheduling/v1/feed/schedule_occurrences.
// startDate and endDate filter on the due time of the occurrences. When one is zero, the API default applies to it
// (5 months back for the start, 1 month ahead for the end).
func NewScheduleOccurrenceV2Feed(templateIDs []string, startDate, endDate time.Time) Feed {
	return &rowFeed[ScheduleOccurrenceV2]{
		name:       "new_schedule_occurrences",
		initialURL: "/scheduling/v1/feed/schedule_occurrences?include_entity_names=true",
		primaryKey: []string{"id", "target_id"},
		order:      "schedule_id, due_time, target_id",
		params: GetFeedParams{
			TemplateIDs: templateIDs,
			StartDate:   startDate,
			EndDate:     endDate,
			Limit:       maxSchedulingFeedLimit,
		},
		key: func(r *ScheduleOccurrenceV2) string { return r.ID + "|" + r.TargetID },
		columns: []string{
			"schedule_id", "occurrence_id", "template_id", "organisation_id", "start_time", "due_time", "miss_time",
			"occurrence_status", "audit_id", "completed_at", "exported_at", "user_id", "assignee_status", "note",
			"assignee_type", "assignee_user_id", "assignee_user_name", "assignee_site_id", "assignee_site_name",
			"assignee_group_id", "assignee_group_name", "completed_by_user_id", "completed_by_user_name", "target_type",
			"target_name", "wont_do_note",
		},
	}
}
