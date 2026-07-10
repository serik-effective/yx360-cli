package disk

// Resource represents a Yandex Disk file or directory resource.
type Resource struct {
	ResourceID string `json:"resource_id,omitempty"`
	Path       string `json:"path"`
	Name       string `json:"name"`
	Type       string `json:"type"` // "file" or "dir"
	Size       int64  `json:"size,omitempty"`
	Created    string `json:"created,omitempty"`
	Modified   string `json:"modified,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
	PublicURL  string `json:"public_url,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
}

// ResourceList is the _embedded section of a directory listing response.
type ResourceList struct {
	Items  []Resource `json:"items"`
	Offset int        `json:"offset"`
	Limit  int        `json:"limit"`
	Total  int        `json:"total,omitempty"`
}

// Link is returned by download/upload/publish endpoints.
type Link struct {
	Href      string `json:"href"`
	Method    string `json:"method"`
	Templated bool   `json:"templated"`
}

// OperationStatus polls async operations (delete, move).
type OperationStatus struct {
	Status string `json:"status"` // "in-progress", "success", "failed"
}

// resourceResponse is the raw API response for resource metadata.
type resourceResponse struct {
	Path      string        `json:"path"`
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Size      int64         `json:"size,omitempty"`
	Created   string        `json:"created,omitempty"`
	Modified  string        `json:"modified,omitempty"`
	MimeType  string        `json:"mime_type,omitempty"`
	PublicURL string        `json:"public_url,omitempty"`
	PublicKey string        `json:"public_key,omitempty"`
	Embedded  *ResourceList `json:"_embedded,omitempty"`
}
