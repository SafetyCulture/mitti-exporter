package feed

// IssueRelation represents a row from the issue_relations feed. It links an issue to the inspection, action or asset it relates to
type IssueRelation struct {
	ID           string `json:"id" csv:"id" gorm:"primarykey;size:375"`
	FromID       string `json:"from_id" csv:"from_id" gorm:"size:128"`
	RelType      string `json:"rel_type" csv:"rel_type" gorm:"size:50"`
	RelCreatedAt string `json:"rel_created_at" csv:"rel_created_at" gorm:"size:40"`
	ToID         string `json:"to_id" csv:"to_id" gorm:"size:128"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (IssueRelation) TableName() string {
	return "issue_relations"
}

// NewIssueRelationFeed returns the feed for the relations between issues and other resources
func NewIssueRelationFeed(limit int) Feed {
	return &rowFeed[IssueRelation]{
		name:       "issue_relations",
		initialURL: "/feed/issue_relations",
		primaryKey: []string{"id"},
		columns:    []string{"from_id", "rel_type", "rel_created_at", "to_id"},
		order:      "from_id, to_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *IssueRelation) string { return r.ID },
	}
}
