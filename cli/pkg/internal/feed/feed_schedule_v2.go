package feed

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

// The V2 schedule feeds use the Scheduling feed service (/scheduling/v1/feed/*) which replaces the legacy
// /feed/schedules* endpoints for organisations on the new Schedules experience.
// They export to separate new_* tables so organisations that are still on the legacy experience keep working.
// Each table has the columns of the legacy table, plus the columns the new Schedules exporter PRD adds.

// maxSchedulingFeedLimit is the largest page size the Scheduling feed service accepts (the default is 50)
const maxSchedulingFeedLimit = 1000

// ScheduleV2 represents a row from the Scheduling feed service schedules feed.
// The legacy schedules columns come first, then the columns the PRD adds
type ScheduleV2 struct {
	ID              string     `json:"id" csv:"schedule_id" gorm:"primarykey;column:schedule_id;size:128"`
	Description     string     `json:"-" csv:"description"`
	Recurrence      string     `json:"recurrence" csv:"recurrence"`
	Duration        string     `json:"duration" csv:"duration" gorm:"size:50"`
	ModifiedAt      time.Time  `json:"modified_at" csv:"modified_at"`
	ExportedAt      time.Time  `json:"-" csv:"exported_at" gorm:"autoUpdateTime"`
	FromDate        *time.Time `json:"-" csv:"from_date"`
	ToDate          *time.Time `json:"-" csv:"to_date"`
	StartTimeHour   int        `json:"-" csv:"start_time_hour"`
	StartTimeMinute int        `json:"-" csv:"start_time_minute"`
	AllMustComplete bool       `json:"-" csv:"all_must_complete"`
	Status          string     `json:"status" csv:"status" gorm:"size:20"`
	OrganisationID  string     `json:"organisation_id" csv:"organisation_id" gorm:"size:37"`
	Timezone        string     `json:"timezone" csv:"timezone"`
	CanLateSubmit   bool       `json:"-" csv:"can_late_submit"`
	SiteID          string     `json:"-" csv:"site_id" gorm:"size:41"`
	TemplateID      string     `json:"template_id" csv:"template_id" gorm:"size:100"`
	CreatorUserID   string     `json:"-" csv:"creator_user_id" gorm:"size:37"`
	AssetID         string     `json:"-" csv:"asset_id" gorm:"size:37"`
	TargetType      string     `json:"-" csv:"target_type" gorm:"size:10"`
	// TargetIDs is the JSON array of the site or asset UUIDs of the schedule, empty for schedules without sites or assets
	TargetIDs JSONArray `json:"-" csv:"target_ids"`
	// TimezoneType is CUSTOM_TIMEZONE or SITE_TIMEZONE. With SITE_TIMEZONE, recurrence has the site wall-clock time
	// as UTC and each site uses its own time zone
	TimezoneType string `json:"timezone_type" csv:"timezone_type" gorm:"size:30"`
}

// UnmarshalJSON decodes the API fields and fills the legacy and target columns. A column that can't be parsed is left empty
func (s *ScheduleV2) UnmarshalJSON(b []byte) error {
	type apiSchedule ScheduleV2
	if err := json.Unmarshal(b, (*apiSchedule)(s)); err != nil {
		return err
	}

	var api struct {
		Title               string   `json:"title"`
		CompletionRule      string   `json:"completion_rule"`
		IsLateSubmitAllowed bool     `json:"is_late_submit_allowed"`
		CreatedBy           string   `json:"created_by"`
		EndingRuleEndDate   string   `json:"ending_rule_end_date"`
		ScheduleType        string   `json:"schedule_type"`
		SiteIDs             []string `json:"site_ids"`
		AssetIDs            []string `json:"asset_ids"`
	}
	if err := json.Unmarshal(b, &api); err != nil {
		return err
	}

	s.Description = api.Title
	s.AllMustComplete = api.CompletionRule == "all"
	s.CanLateSubmit = api.IsLateSubmitAllowed
	s.CreatorUserID = api.CreatedBy
	if len(api.SiteIDs) == 1 {
		s.SiteID = api.SiteIDs[0]
	}
	if len(api.AssetIDs) == 1 {
		s.AssetID = api.AssetIDs[0]
	}

	var targetIDs []string
	switch api.ScheduleType {
	case "SCHEDULE_TYPE_SITES":
		s.TargetType = "SITE"
		targetIDs = siteUUIDs(api.SiteIDs)
	case "SCHEDULE_TYPE_ASSETS":
		s.TargetType = "ASSET"
		targetIDs = api.AssetIDs
	}
	if s.TargetType != "" {
		ids, err := json.Marshal(targetIDs)
		if err != nil {
			return err
		}
		s.TargetIDs = JSONArray(ids)
	}

	if toDate, err := time.Parse(time.RFC3339, api.EndingRuleEndDate); err == nil {
		s.ToDate = &toDate
	}
	if fromDate := parseDTStart(s.Recurrence, s.Timezone); fromDate != nil {
		// The start time is the local time of the schedule, and the date is in UTC like the other times
		s.StartTimeHour = fromDate.Hour()
		s.StartTimeMinute = fromDate.Minute()
		utc := fromDate.UTC()
		s.FromDate = &utc
	}
	return nil
}

// siteUUIDs converts site IDs from the location_<hex> form to UUIDs, the form of target_id.
// An ID that isn't in that form is kept as it is.
func siteUUIDs(siteIDs []string) []string {
	result := make([]string, len(siteIDs))
	for i, siteID := range siteIDs {
		result[i] = siteID
		hex, found := strings.CutPrefix(siteID, "location_")
		if !found || len(hex) != 32 {
			continue
		}
		if u, err := uuid.FromString(hex); err == nil {
			result[i] = u.String()
		}
	}
	return result
}

// parseDTStart returns the DTSTART of a recurrence in its local time zone, or nil when there is none or it can't be parsed.
// A DTSTART without TZID or a Z suffix is in the time zone of the schedule.
func parseDTStart(recurrence, timezone string) *time.Time {
	for _, line := range strings.Split(recurrence, "\n") {
		rest, found := strings.CutPrefix(strings.TrimSpace(line), "DTSTART")
		if !found {
			continue
		}
		params, value, ok := strings.Cut(rest, ":")
		if !ok {
			return nil
		}

		tzName := timezone
		if tzid, found := strings.CutPrefix(params, ";TZID="); found {
			tzName = tzid
		}
		loc, err := time.LoadLocation(tzName)
		if err != nil {
			return nil
		}

		layout := "20060102T150405"
		if strings.HasSuffix(value, "Z") {
			layout += "Z"
			loc = time.UTC
		}
		t, err := time.ParseInLocation(layout, value, loc)
		if err != nil {
			return nil
		}
		return &t
	}
	return nil
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (ScheduleV2) TableName() string {
	return "new_schedules"
}

// withScheduleStatuses adds the status filter of the schedules and schedule assignees feeds to their path.
// No statuses, or none that are known, keeps the path so schedules of every status are returned
func withScheduleStatuses(path string, statuses []string) string {
	show := map[string]bool{}
	for _, status := range statuses {
		show[strings.ToUpper(strings.TrimSpace(status))] = true
	}
	if !show["ACTIVE"] && !show["PAUSED"] && !show["FINISHED"] {
		return path
	}

	query := url.Values{}
	query.Set("show_active", strconv.FormatBool(show["ACTIVE"]))
	query.Set("show_paused", strconv.FormatBool(show["PAUSED"]))
	query.Set("show_finished", strconv.FormatBool(show["FINISHED"]))
	return path + "?" + query.Encode()
}

// NewScheduleV2Feed returns the feed for /scheduling/v1/feed/schedules.
// statuses filters on the schedule status (ACTIVE, PAUSED, FINISHED); it is empty for every status
func NewScheduleV2Feed(templateIDs []string, statuses []string) Feed {
	return &rowFeed[ScheduleV2]{
		name:       "new_schedules",
		initialURL: withScheduleStatuses("/scheduling/v1/feed/schedules", statuses),
		primaryKey: []string{"schedule_id"},
		order:      "schedule_id",
		params: GetFeedParams{
			TemplateIDs: templateIDs,
			Limit:       maxSchedulingFeedLimit,
		},
		key: func(r *ScheduleV2) string { return r.ID },
		columns: []string{
			"description", "recurrence", "duration", "modified_at", "exported_at", "from_date", "to_date",
			"start_time_hour", "start_time_minute", "all_must_complete", "status", "organisation_id", "timezone",
			"can_late_submit", "site_id", "template_id", "creator_user_id", "asset_id", "target_type", "target_ids",
			"timezone_type",
		},
	}
}
