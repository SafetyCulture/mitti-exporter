package api_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
)

func TestCSVExporterSupportsUpsert_should_return_true(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	assert.True(t, exporter.SupportsUpsert())
}

func TestCSVExporterInitFeed_should_create_table_if_not_exists(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	// This query will only work for SQLite
	var result []struct {
		Name string
	}

	resp := exporter.DB.Raw("SELECT name FROM sqlite_master WHERE type='table';").Scan(&result)
	assert.Nil(t, resp.Error)

	assert.Len(t, result, 1)
	assert.Equal(t, "users", result[0].Name)
}

func TestCSVExporterInitFeed_should_truncate_table_if_truncate_is_true(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	users := []feed.User{
		{
			ID:        "user_1",
			Firstname: "User One",
			Lastname:  "User One",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: true,
	})
	assert.NoError(t, err)

	var rowCount int64
	resp := exporter.DB.Table("users").Count(&rowCount)
	assert.Nil(t, resp.Error)
	assert.Equal(t, int64(0), rowCount)
}

func TestCSVExporterInitFeed_should_not_truncate_table_if_truncate_is_false(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	users := []feed.User{
		{
			ID:        "user_1",
			Firstname: "User One",
			Lastname:  "User One",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	var rowCount int64
	resp := exporter.DB.Table("users").Count(&rowCount)
	assert.Nil(t, resp.Error)
	assert.Equal(t, int64(1), rowCount)
}

func TestCSVExporterWriteRows_should_write_rows(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	users := []feed.User{
		{
			ID:        "user_1",
			Firstname: "User 1",
			Lastname:  "User 1",
		},
		{
			ID:        "user_2",
			Firstname: "User 2",
			Lastname:  "User 2",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	var rows []feed.User
	resp := exporter.DB.Table("users").Scan(&rows)
	assert.Nil(t, resp.Error)

	assert.Equal(t, 2, len(rows))
	assert.Equal(t, "user_1", rows[0].ID)
	assert.Equal(t, "user_2", rows[1].ID)
}

func TestCSVExporterWriteRows_should_update_rows(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: true,
	})
	assert.NoError(t, err)

	users := []feed.User{
		{
			ID:        "user_1",
			Firstname: "User 1",
			Lastname:  "User 1",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	users = []feed.User{
		{
			ID:        "user_1",
			Firstname: "User One",
			Lastname:  "User One",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	var rows []feed.User
	resp := exporter.DB.Table("users").Scan(&rows)
	assert.Nil(t, resp.Error)

	assert.Equal(t, 1, len(rows))
	assert.Equal(t, "user_1", rows[0].ID)
	assert.Equal(t, "User One", rows[0].Firstname)
	assert.Equal(t, "User One", rows[0].Lastname)
}

func TestCSVExporterLastModifiedAt_should_return_latest_modified_at(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	inspectionFeed := &feed.InspectionFeed{}

	err = exporter.InitFeed(inspectionFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	now := time.Now()
	inspections := []feed.Inspection{
		{
			ID:         "audit_1",
			ModifiedAt: now,
		},
		{
			ID:         "audit_2",
			ModifiedAt: time.Now().Add(time.Hour * -128),
		},
		{
			ID:         "audit_3",
			ModifiedAt: time.Now().Add(time.Hour * -3000),
		},
		{
			ID:         "audit_4",
			ModifiedAt: time.Now().Add(time.Hour * -2),
		},
	}

	err = exporter.WriteRows(inspectionFeed, inspections)
	assert.NoError(t, err)

	// Check the timestamp for the audits that doesn't have organisation_id
	lastModifiedAt, err := exporter.LastModifiedAt(inspectionFeed, time.Now().Add(time.Hour*-30000), "role_123")
	assert.NoError(t, err)
	// Times are slightly lossy, convert to ISO string
	assert.Equal(t, now.Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))

	lastModifiedAt, err = exporter.LastModifiedAt(inspectionFeed, time.Now().Add(time.Hour*-30000), "role_1234")
	assert.NoError(t, err)
	// Times are slightly lossy, convert to ISO string
	assert.Equal(t, now.Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))

	inspections = []feed.Inspection{
		{
			ID:             "audit_5",
			ModifiedAt:     now,
			OrganisationID: "role_123",
		},
		{
			ID:             "audit_6",
			ModifiedAt:     now.Add(time.Hour * -128),
			OrganisationID: "role_123",
		},
		{
			ID:             "audit_7",
			ModifiedAt:     now.Add(time.Hour * -3000),
			OrganisationID: "role_1234",
		},
		{
			ID:             "audit_8",
			ModifiedAt:     now.Add(time.Hour * -2),
			OrganisationID: "role_1234",
		},
	}

	err = exporter.WriteRows(inspectionFeed, inspections)
	assert.NoError(t, err)

	// Check the timestamp for the audits that contains organisation_id
	lastModifiedAt, err = exporter.LastModifiedAt(inspectionFeed, time.Now().Add(time.Hour*-30000), "role_123")
	assert.NoError(t, err)
	// Times are slightly lossy, convert to ISO string
	assert.Equal(t, now.Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))

	lastModifiedAt, err = exporter.LastModifiedAt(inspectionFeed, time.Now().Add(time.Hour*-30000), "role_1234")
	assert.NoError(t, err)
	// Times are slightly lossy, convert to ISO string
	assert.Equal(t, now.Add(time.Hour*-2).Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))
}

func TestCSVExporterLastModifiedAt_should_return_modified_after_if_latest(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	inspectionFeed := &feed.InspectionFeed{}

	err = exporter.InitFeed(inspectionFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	now := time.Now()
	inspections := []feed.Inspection{
		{
			ID:         "audit_1",
			ModifiedAt: now,
		},
		{
			ID:         "audit_2",
			ModifiedAt: now.Add(time.Hour * -128),
		},
		{
			ID:         "audit_3",
			ModifiedAt: now.Add(time.Hour * -3000),
		},
		{
			ID:         "audit_4",
			ModifiedAt: now.Add(time.Hour * -2),
		},
	}

	err = exporter.WriteRows(inspectionFeed, inspections)
	assert.NoError(t, err)

	// Check the timestamp for the audits that doesn't have organisation_id
	lastModifiedAt, err := exporter.LastModifiedAt(inspectionFeed, now.Add(time.Hour), "role_123")
	assert.NoError(t, err)
	// Times are slightly lossy, converting to ISO string
	assert.Equal(t, now.Add(time.Hour).Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))

	lastModifiedAt, err = exporter.LastModifiedAt(inspectionFeed, now.Add(time.Hour), "role_124")
	assert.NoError(t, err)
	// Times are slightly lossy, converting to ISO string
	assert.Equal(t, now.Add(time.Hour).Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))

	inspections = []feed.Inspection{
		{
			ID:             "audit_5",
			ModifiedAt:     now,
			OrganisationID: "role_123",
		},
		{
			ID:             "audit_6",
			ModifiedAt:     now.Add(time.Hour * -128),
			OrganisationID: "role_123",
		},
		{
			ID:             "audit_7",
			ModifiedAt:     now.Add(time.Hour * -3000),
			OrganisationID: "role_1234",
		},
		{
			ID:             "audit_8",
			ModifiedAt:     now.Add(time.Hour * -2),
			OrganisationID: "role_1234",
		},
	}

	err = exporter.WriteRows(inspectionFeed, inspections)
	assert.NoError(t, err)

	// Check the timestamp for the audits that contains organisation_id
	lastModifiedAt, err = exporter.LastModifiedAt(inspectionFeed, now.Add(time.Hour), "role_123")
	assert.NoError(t, err)
	// Times are slightly lossy, converting to ISO string
	assert.Equal(t, now.Add(time.Hour).Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))

	lastModifiedAt, err = exporter.LastModifiedAt(inspectionFeed, now.Add(time.Hour), "role_124")
	assert.NoError(t, err)
	// Times are slightly lossy, converting to ISO string
	assert.Equal(t, now.Add(time.Hour).Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))
}

func TestCSVExporter_should_fail_if_path_is_wrong(t *testing.T) {
	exporter, err := feed.NewCSVExporter("/xyz", "", 1)
	require.Nil(t, exporter)
	require.NotNil(t, err)
}

func TestCSVExporter_should_do_rollover_files(t *testing.T) {
	exporter, err := getTemporaryCSVExporterWithMaxRowsLimit(1)
	require.Nil(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	users := []feed.User{
		{
			ID:             "user_1",
			OrganisationID: "role_123",
			Email:          "user.1@example.com",
			Firstname:      "User 1",
			Lastname:       "User 1",
		},
		{
			ID:             "user_2",
			OrganisationID: "role_123",
			Email:          "user.2@example.com",
			Firstname:      "User 2",
			Lastname:       "User 2",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	err = exporter.FinaliseExport(userFeed, &[]feed.User{})
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(exporter.ExportPath, "users.csv"))
	assert.NoError(t, err)

	contentString := dateRegex.ReplaceAllLiteralString(strings.TrimSpace(string(content)), "--date--")

	expected := `user_id,organisation_id,email,firstname,lastname,active,last_seen_at,exported_at,seat_type,created_at
user_2,role_123,user.2@example.com,User 2,User 2,false,,--date--,,--date--`
	assert.Equal(t, strings.TrimSpace(expected), contentString)
}

func TestCSVExporterFinaliseExport_should_write_rows_out_to_file(t *testing.T) {
	exporter, err := getTemporaryCSVExporter()
	assert.NoError(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	users := []feed.User{
		{
			ID:             "user_1",
			OrganisationID: "role_123",
			Email:          "user.1@example.com",
			Firstname:      "User 1",
			Lastname:       "User 1",
		},
		{
			ID:             "user_2",
			OrganisationID: "role_123",
			Email:          "user.2@example.com",
			Firstname:      "User 2",
			Lastname:       "User 2",
			SeatType:       "premium",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	users = []feed.User{
		{
			ID:             "user_1",
			OrganisationID: "role_123",
			Email:          "user.1@example.com",
			Firstname:      "User 1",
			Lastname:       "User 1",
		},
		{
			ID:             "user_2",
			OrganisationID: "role_123",
			Email:          "user.2@example.com",
			Firstname:      "User 2",
			Lastname:       "User 2",
			SeatType:       "premium",
		},
		{
			ID:             "user_3",
			OrganisationID: "role_123",
			Email:          "user.3@example.com",
			Firstname:      "User 3",
			Lastname:       "User 3",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	err = exporter.FinaliseExport(userFeed, &[]feed.User{})
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(exporter.ExportPath, "users.csv"))
	assert.NoError(t, err)

	contentString := dateRegex.ReplaceAllLiteralString(strings.TrimSpace(string(content)), "--date--")

	expected := `user_id,organisation_id,email,firstname,lastname,active,last_seen_at,exported_at,seat_type,created_at
user_1,role_123,user.1@example.com,User 1,User 1,false,,--date--,,--date--
user_2,role_123,user.2@example.com,User 2,User 2,false,,--date--,premium,--date--
user_3,role_123,user.3@example.com,User 3,User 3,false,,--date--,,--date--`
	assert.Equal(t, strings.TrimSpace(expected), contentString)
}

func TestCSVExporterFinaliseExport_should_write_rows_to_multiple_file(t *testing.T) {
	exporter, err := getTemporaryCSVExporterWithMaxRowsLimit(2)
	assert.NoError(t, err)

	userFeed := &feed.UserFeed{}

	err = exporter.InitFeed(userFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	users := []feed.User{
		{
			ID:             "user_1",
			OrganisationID: "role_123",
			Email:          "user.1@example.com",
			Firstname:      "User 1",
			Lastname:       "User 1",
		},
		{
			ID:             "user_2",
			OrganisationID: "role_123",
			Email:          "user.2@example.com",
			Firstname:      "User 2",
			Lastname:       "User 2",
		},
		{
			ID:             "user_3",
			OrganisationID: "role_123",
			Email:          "user.3@example.com",
			Firstname:      "User 3",
			Lastname:       "User 3",
			SeatType:       "premium",
		},
	}

	err = exporter.WriteRows(userFeed, users)
	assert.NoError(t, err)

	err = exporter.FinaliseExport(userFeed, &[]feed.User{})
	assert.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(exporter.ExportPath, "users*.csv"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(files))

	content1, err := os.ReadFile(files[0])
	assert.NoError(t, err)

	content1String := dateRegex.ReplaceAllLiteralString(strings.TrimSpace(string(content1)), "--date--")

	expected1 := `user_id,organisation_id,email,firstname,lastname,active,last_seen_at,exported_at,seat_type,created_at
user_1,role_123,user.1@example.com,User 1,User 1,false,,--date--,,--date--
user_2,role_123,user.2@example.com,User 2,User 2,false,,--date--,,--date--`
	assert.Equal(t, strings.TrimSpace(expected1), content1String)

	content2, err := os.ReadFile(files[1])
	assert.NoError(t, err)

	content2String := dateRegex.ReplaceAllLiteralString(strings.TrimSpace(string(content2)), "--date--")

	expected2 := `user_id,organisation_id,email,firstname,lastname,active,last_seen_at,exported_at,seat_type,created_at
user_3,role_123,user.3@example.com,User 3,User 3,false,,--date--,premium,--date--`
	assert.Equal(t, strings.TrimSpace(expected2), content2String)
}
