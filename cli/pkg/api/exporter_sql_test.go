package api_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/feed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLExporterSupportsUpsert_should_return_true(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	assert.True(t, exporter.SupportsUpsert())
}

func TestSQLExporterInitFeed_should_create_table_if_not_exists(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
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

func TestSQLExporterInitFeed_should_truncate_table_if_truncate_is_true(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
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

func TestSQLExporterInitFeed_should_not_truncate_table_if_truncate_is_false(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
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

func TestSQLExporterWriteRows_should_write_rows(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
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

func TestSQLExporter_WriteRows_should_upsert_when_pk_conflict(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	require.Nil(t, err)
	require.NotNil(t, exporter)

	groupUserFeed := feed.GroupUserFeed{}
	feedOptions := feed.InitFeedOptions{
		Truncate: false,
	}

	err = exporter.InitFeed(&groupUserFeed, &feedOptions)
	require.Nil(t, err)

	feedData := []feed.GroupUser{
		{UserID: "UID_1", GroupID: "GID_1", OrganisationID: "OID_1"},
		{UserID: "UID_2", GroupID: "GID_1", OrganisationID: "OID_1"},
		{UserID: "UID_3", GroupID: "GID_1", OrganisationID: "OID_1"},
		{UserID: "UID_4", GroupID: "GID_2", OrganisationID: "OID_1"},
		{UserID: "UID_1", GroupID: "GID_1", OrganisationID: "OID_1b"}, // should upsert there
	}
	err = exporter.WriteRows(&groupUserFeed, feedData)
	require.Nil(t, err)

	var dbData []feed.GroupUser
	sqlRes := exporter.DB.Table("group_users").Scan(&dbData)
	assert.Nil(t, sqlRes.Error)
	assert.EqualValues(t, 4, len(dbData))
	assert.EqualValues(t, "UID_1", dbData[0].UserID)
	assert.EqualValues(t, "GID_1", dbData[0].GroupID)
	assert.EqualValues(t, "OID_1b", dbData[0].OrganisationID)
}

func TestSQLExporterWriteRows_should_update_rows(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
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

func TestSQLExporterLastModifiedAt_should_return_latest_modified_at(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
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

func TestSQLExporterLastModifiedAt_should_return_modified_after_if_latest(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
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

func TestSQLExporterLastRecord_should_return_modified_after_if_latest(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	inspectionFeed := &feed.InspectionFeed{}

	err = exporter.InitFeed(inspectionFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	now := time.Now()
	inspections := []feed.Inspection{
		{
			ID:             "audit_1",
			ModifiedAt:     now.Add(time.Hour * -1),
			OrganisationID: "role_123",
		},
		{
			ID:             "audit_2",
			ModifiedAt:     now.Add(time.Hour * -128),
			OrganisationID: "role_123",
		},
		{
			ID:             "audit_3",
			ModifiedAt:     now.Add(time.Hour * -3000),
			OrganisationID: "role_123",
		},
		{
			ID:             "audit_4",
			ModifiedAt:     now.Add(time.Hour * -2),
			OrganisationID: "role_123",
		},
	}

	err = exporter.WriteRows(inspectionFeed, inspections)
	assert.NoError(t, err)

	// Check the timestamp for the audits that doesn't have organisation_id
	lastModifiedAt := exporter.LastRecord(inspectionFeed, now.Add(time.Hour), "role_123", "modified_at")
	assert.Equal(t, now.Add(time.Hour*-1).Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))
}

func TestSQLExporterLastRecord_should_return_the_given_date_when_no_rows_matching(t *testing.T) {
	exporter, err := getInmemorySQLExporter("")
	assert.NoError(t, err)

	inspectionFeed := &feed.InspectionFeed{}

	err = exporter.InitFeed(inspectionFeed, &feed.InitFeedOptions{
		Truncate: false,
	})
	assert.NoError(t, err)

	now := time.Now()
	inspections := []feed.Inspection{
		{
			ID:             "audit_1",
			ModifiedAt:     now.Add(time.Hour * -1),
			OrganisationID: "role_1234",
		},
		{
			ID:             "audit_2",
			ModifiedAt:     now.Add(time.Hour * -128),
			OrganisationID: "role_1234",
		},
		{
			ID:             "audit_3",
			ModifiedAt:     now.Add(time.Hour * -3000),
			OrganisationID: "role_1234",
		},
		{
			ID:             "audit_4",
			ModifiedAt:     now.Add(time.Hour * -2),
			OrganisationID: "role_1234",
		},
	}

	err = exporter.WriteRows(inspectionFeed, inspections)
	assert.NoError(t, err)

	// Check the timestamp for the audits that doesn't have organisation_id
	lastModifiedAt := exporter.LastRecord(inspectionFeed, now.Add(time.Hour), "role_123", "modified_at")
	assert.Equal(t, now.Add(time.Hour).Format(time.RFC3339), lastModifiedAt.Format(time.RFC3339))
}

func TestNewSQLExporter_should_create_exporter_for_sqlite(t *testing.T) {
	sqlExporter, err := feed.NewSQLExporter("sqlite", "file::memory:", true, "")
	assert.NoError(t, err)

	assert.NotNil(t, sqlExporter)
}

func TestNewSQLExporter_should_return_error_for_invalid_dialect(t *testing.T) {
	sqlExporter, err := feed.NewSQLExporter("not-supported", "file::memory:", true, "")
	assert.NotNil(t, err)
	assert.Nil(t, sqlExporter)
}

func TestNewSQLExporter_should_return_error_for_connection_errors(t *testing.T) {
	sqlExporter, err := feed.NewSQLExporter("sqlite", "*$////bad connection string", true, "")
	assert.NotNil(t, err)
	assert.Nil(t, sqlExporter)
}

func TestSQLExporterWriteMedia(t *testing.T) {
	dir, err := os.MkdirTemp("", "export")
	if err != nil {
		log.Fatal(err)
	}

	sqlExporter, err := feed.NewSQLExporter("sqlite", "file::memory:", true, dir)
	assert.NoError(t, err)

	sqlExporter.WriteMedia("1234", "12345", "image/jpeg", []byte("sample-string"))
}
