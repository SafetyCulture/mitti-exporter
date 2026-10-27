package feed

// CredentialType represents a row from the credential_types feed
type CredentialType struct {
	DocumentTypeID   string `json:"document_type_id" csv:"document_type_id" gorm:"primarykey;size:128"`
	DocumentTypeName string `json:"document_type_name" csv:"document_type_name"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (CredentialType) TableName() string {
	return "credential_types"
}

// Credential represents a row from the credentials feed
type Credential struct {
	DocumentID           string `json:"document_id" csv:"document_id" gorm:"primarykey;size:128"`
	DocumentVersionID    string `json:"document_version_id" csv:"document_version_id" gorm:"size:128"`
	DocumentTypeID       string `json:"document_type_id" csv:"document_type_id" gorm:"size:128"`
	DocumentTypeName     string `json:"document_type_name" csv:"document_type_name"`
	SubjectUserID        string `json:"subject_user_id" csv:"subject_user_id" gorm:"size:128"`
	SubjectUserFirstName string `json:"subject_user_first_name" csv:"subject_user_first_name"`
	SubjectUserLastName  string `json:"subject_user_last_name" csv:"subject_user_last_name"`
	IssueDate            string `json:"issue_date" csv:"issue_date" gorm:"size:40"`
	ExpiryDate           string `json:"expiry_date" csv:"expiry_date" gorm:"size:40"`
	ExpiryStatus         string `json:"expiry_status" csv:"expiry_status" gorm:"size:50"`
	ApprovalStatus       string `json:"approval_status" csv:"approval_status" gorm:"size:50"`
	CreatedAt            string `json:"created_at" csv:"created_at" gorm:"size:40"`
	ModifiedAt           string `json:"modified_at" csv:"modified_at" gorm:"size:40"`
	Deleted              bool   `json:"deleted" csv:"deleted"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (Credential) TableName() string {
	return "credentials"
}

// NewCredentialTypeFeed returns the feed for /credentials/v1/feed/credential-types
func NewCredentialTypeFeed(limit int) Feed {
	return &rowFeed[CredentialType]{
		name:       "credential_types",
		initialURL: "/credentials/v1/feed/credential-types",
		primaryKey: []string{"document_type_id"},
		columns:    []string{"document_type_name"},
		order:      "document_type_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *CredentialType) string { return r.DocumentTypeID },
	}
}

// NewCredentialFeed returns the feed for /credentials/v1/feed/credentials.
// The feed is always exported in full, so credentials deleted upstream are not part of the export
func NewCredentialFeed(limit int) Feed {
	return &rowFeed[Credential]{
		name:       "credentials",
		initialURL: "/credentials/v1/feed/credentials",
		primaryKey: []string{"document_id"},
		order:      "document_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *Credential) string { return r.DocumentID },
		columns: []string{
			"document_version_id", "document_type_id", "document_type_name", "subject_user_id", "subject_user_first_name",
			"subject_user_last_name", "issue_date", "expiry_date", "expiry_status", "approval_status", "created_at",
			"modified_at", "deleted",
		},
	}
}
