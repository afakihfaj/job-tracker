package models

import "time"

const (
	StatusApplied   = "Applied"
	StatusInterview = "Interview"
	StatusOffering  = "Offering"
	StatusContract  = "TTD Kontrak"
	StatusRejected  = "Rejected"
)

var ValidStatuses = map[string]bool{
	StatusApplied:   true,
	StatusInterview: true,
	StatusOffering:  true,
	StatusContract:  true,
	StatusRejected:  true,
}

func IsValidStatus(status string) bool {
	return ValidStatuses[status]
}

type Application struct {
	ID        int64     `json:"id"`
	Company   string    `json:"company"`
	Role      string    `json:"role"`
	Platform  string    `json:"platform"`
	Status    string    `json:"status"`
	Salary    int64     `json:"salary"`
	Notes     string    `json:"notes"`
	AppliedAt time.Time `json:"applied_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StatusLog struct {
	ID            int64     `json:"id"`
	ApplicationID int64     `json:"application_id"`
	OldStatus     string    `json:"old_status"`
	NewStatus     string    `json:"new_status"`
	ChangedAt     time.Time `json:"changed_at"`
}

type CreateApplicationRequest struct {
	Company   string `json:"company"`
	Role      string `json:"role"`
	Platform  string `json:"platform"`
	Status    string `json:"status"`
	Salary    int64  `json:"salary"`
	Notes     string `json:"notes"`
	AppliedAt string `json:"applied_at"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}
