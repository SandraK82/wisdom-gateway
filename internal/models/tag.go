package models

import "time"

// TagCategory classifies tags for better organization and filtering.
type TagCategory string

const (
	TagCategoryPlatform     TagCategory = "PLATFORM"
	TagCategoryLanguage     TagCategory = "LANGUAGE"
	TagCategoryFramework    TagCategory = "FRAMEWORK"
	TagCategoryLibrary      TagCategory = "LIBRARY"
	TagCategoryVersion      TagCategory = "VERSION"
	TagCategoryDomain       TagCategory = "DOMAIN"
	TagCategoryType         TagCategory = "TYPE"
	TagCategoryEnvironment  TagCategory = "ENVIRONMENT"
	TagCategoryArchitecture TagCategory = "ARCHITECTURE"
	TagCategoryCountry      TagCategory = "COUNTRY"
	TagCategoryField        TagCategory = "FIELD"
)

// Tag represents a classification label for fragments.
type Tag struct {
	UUID      string      `json:"uuid"`
	Name      string      `json:"name"`      // Unique, normalized name
	Content   string      `json:"content"`   // Description of the tag
	Version   uint32      `json:"version"`   // Incremented on updates
	Category  TagCategory `json:"category"`  // Classification category
	Creator   Address     `json:"creator"`   // Agent who created this tag
	Signature string      `json:"signature"` // Signature over the tag data
	CreatedAt time.Time   `json:"created_at"` // Database creation timestamp
}

// Validate checks if the Tag data is well-formed.
func (t *Tag) Validate() error {
	if t.UUID == "" {
		return NewValidationError("tag", "uuid is required")
	}
	if t.Name == "" {
		return NewValidationError("tag", "name is required")
	}
	if t.Category == "" {
		return NewValidationError("tag", "category is required")
	}
	if !isValidTagCategory(t.Category) {
		return NewValidationError("tag", "invalid category: %s", t.Category)
	}
	if t.Creator.IsEmpty() {
		return NewValidationError("tag", "creator is required")
	}
	if err := t.Creator.Validate(); err != nil {
		return NewValidationError("tag", "invalid creator: %v", err)
	}
	if t.Signature == "" {
		return NewValidationError("tag", "signature is required")
	}
	
	return nil
}

// isValidTagCategory checks if a tag category is valid.
func isValidTagCategory(tc TagCategory) bool {
	switch tc {
	case TagCategoryPlatform, TagCategoryLanguage, TagCategoryFramework,
		TagCategoryLibrary, TagCategoryVersion, TagCategoryDomain,
		TagCategoryType, TagCategoryEnvironment, TagCategoryArchitecture,
		TagCategoryCountry, TagCategoryField:
		return true
	}
	return false
}

// ValidTagCategories returns all valid tag categories.
func ValidTagCategories() []TagCategory {
	return []TagCategory{
		TagCategoryPlatform, TagCategoryLanguage, TagCategoryFramework,
		TagCategoryLibrary, TagCategoryVersion, TagCategoryDomain,
		TagCategoryType, TagCategoryEnvironment, TagCategoryArchitecture,
		TagCategoryCountry, TagCategoryField,
	}
}
