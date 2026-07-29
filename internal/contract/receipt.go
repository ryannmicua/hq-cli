package contract

type Receipt struct {
	SchemaVersion     string `json:"schemaVersion"`
	Cursor            uint64 `json:"cursor"`
	RequestID         string `json:"requestId"`
	Target            string `json:"target"`
	TargetSha256      string `json:"targetSha256"`
	RenderedSha256    string `json:"renderedSha256"`
	BackupPath        string `json:"backupPath,omitempty"`
	BackupSha256      string `json:"backupSha256,omitempty"`
	ApprovalReference string `json:"approvalReference,omitempty"`
	AppliedAt         string `json:"appliedAt"`
}
