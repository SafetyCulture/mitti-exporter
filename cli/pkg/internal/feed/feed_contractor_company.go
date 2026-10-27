package feed

// ContractorCompany represents a row from the contractor_companies feed
type ContractorCompany struct {
	CompanyID                 string `json:"company_id" csv:"company_id" gorm:"primarykey;size:128"`
	CompanyName               string `json:"company_name" csv:"company_name"`
	CompanyTypeID             string `json:"company_type_id" csv:"company_type_id" gorm:"size:128"`
	CompanyTypeName           string `json:"company_type_name" csv:"company_type_name"`
	Email                     string `json:"email" csv:"email"`
	PhoneNumber               string `json:"phone_number" csv:"phone_number"`
	AddressSubPremise         string `json:"address_sub_premise" csv:"address_sub_premise"`
	AddressStreetNumber       string `json:"address_street_number" csv:"address_street_number"`
	AddressStreet             string `json:"address_street" csv:"address_street"`
	AddressCity               string `json:"address_city" csv:"address_city"`
	AddressAdministrativeArea string `json:"address_administrative_area" csv:"address_administrative_area"`
	AddressPostalCode         string `json:"address_postal_code" csv:"address_postal_code"`
	AddressCountryCode        string `json:"address_country_code" csv:"address_country_code" gorm:"size:10"`
	ExternalReferenceID       string `json:"external_reference_id" csv:"external_reference_id"`
	Status                    string `json:"status" csv:"status" gorm:"size:50"`
	CreatedAt                 string `json:"created_at" csv:"created_at" gorm:"size:40"`
	ModifiedAt                string `json:"modified_at" csv:"modified_at" gorm:"size:40"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (ContractorCompany) TableName() string {
	return "contractor_companies"
}

// ContractorCompanyDocument represents a row from the contractor_company_documents feed
type ContractorCompanyDocument struct {
	CompanyID          string `json:"company_id" csv:"company_id" gorm:"primarykey;size:128"`
	DocumentID         string `json:"document_id" csv:"document_id" gorm:"primarykey;size:128"`
	DocumentVersion    string `json:"document_version" csv:"document_version" gorm:"primarykey;size:50"`
	CompanyName        string `json:"company_name" csv:"company_name"`
	DocumentTypeID     string `json:"document_type_id" csv:"document_type_id" gorm:"size:128"`
	DocumentTypeName   string `json:"document_type_name" csv:"document_type_name"`
	DocumentStatus     string `json:"document_status" csv:"document_status" gorm:"size:50"`
	DocumentIssueDate  string `json:"document_issue_date" csv:"document_issue_date" gorm:"size:40"`
	DocumentExpiryDate string `json:"document_expiry_date" csv:"document_expiry_date" gorm:"size:40"`
	DocumentName       string `json:"document_name" csv:"document_name"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (ContractorCompanyDocument) TableName() string {
	return "contractor_company_documents"
}

// ContractorCompanyMember represents a row from the contractor_company_members feed
type ContractorCompanyMember struct {
	CompanyID    string `json:"company_id" csv:"company_id" gorm:"primarykey;size:128"`
	UserID       string `json:"user_id" csv:"user_id" gorm:"primarykey;size:128"`
	CompanyName  string `json:"company_name" csv:"company_name"`
	UserUsername string `json:"user_username" csv:"user_username"`
	UserRole     string `json:"user_role" csv:"user_role" gorm:"size:100"`
}

// TableName makes the table match the name of the feed. By default gorm derives it from the type name
func (ContractorCompanyMember) TableName() string {
	return "contractor_company_members"
}

// NewContractorCompanyFeed returns the feed for /companies/v1/feed/companies
func NewContractorCompanyFeed(limit int) Feed {
	return &rowFeed[ContractorCompany]{
		name:       "contractor_companies",
		initialURL: "/companies/v1/feed/companies",
		primaryKey: []string{"company_id"},
		order:      "company_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *ContractorCompany) string { return r.CompanyID },
		columns: []string{
			"company_name", "company_type_id", "company_type_name", "email", "phone_number", "address_sub_premise",
			"address_street_number", "address_street", "address_city", "address_administrative_area",
			"address_postal_code", "address_country_code", "external_reference_id", "status", "created_at", "modified_at",
		},
	}
}

// NewContractorCompanyDocumentFeed returns the feed for /companies/v1/feed/companies/documents
func NewContractorCompanyDocumentFeed(limit int) Feed {
	return &rowFeed[ContractorCompanyDocument]{
		name:       "contractor_company_documents",
		initialURL: "/companies/v1/feed/companies/documents",
		primaryKey: []string{"company_id", "document_id", "document_version"},
		order:      "company_id, document_id, document_version",
		params:     GetFeedParams{Limit: limit},
		key: func(r *ContractorCompanyDocument) string {
			return r.CompanyID + "|" + r.DocumentID + "|" + r.DocumentVersion
		},
		columns: []string{
			"company_name", "document_type_id", "document_type_name", "document_status", "document_issue_date",
			"document_expiry_date", "document_name",
		},
	}
}

// NewContractorCompanyMemberFeed returns the feed for /companies/v1/feed/companies/members
func NewContractorCompanyMemberFeed(limit int) Feed {
	return &rowFeed[ContractorCompanyMember]{
		name:       "contractor_company_members",
		initialURL: "/companies/v1/feed/companies/members",
		primaryKey: []string{"company_id", "user_id"},
		columns:    []string{"company_name", "user_username", "user_role"},
		order:      "company_id, user_id",
		params:     GetFeedParams{Limit: limit},
		key:        func(r *ContractorCompanyMember) string { return r.CompanyID + "|" + r.UserID },
	}
}
