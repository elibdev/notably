package helpers

import (
	"time"

	"github.com/elibdev/notably/internal/models"
)

// Test Users
var (
	TestUser1 = models.NewUser("user_123")
	TestUser2 = models.NewUser("user_456")
	TestUser3 = models.NewUser("admin_user")
)

// Test Table Schemas
var (
	ContactsFields = []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
		{Name: "email", DataType: models.DataTypeString},
		{Name: "phone", DataType: models.DataTypeString},
		{Name: "age", DataType: models.DataTypeInt},
		{Name: "is_active", DataType: models.DataTypeBool},
		{Name: "created_date", DataType: models.DataTypeDate},
	}

	NotesFields = []models.FieldDefinition{
		{Name: "title", DataType: models.DataTypeString},
		{Name: "content", DataType: models.DataTypeString},
		{Name: "tags", DataType: models.DataTypeJSON},
		{Name: "is_public", DataType: models.DataTypeBool},
		{Name: "word_count", DataType: models.DataTypeInt},
	}

	ProjectsFields = []models.FieldDefinition{
		{Name: "name", DataType: models.DataTypeString},
		{Name: "description", DataType: models.DataTypeString},
		{Name: "status", DataType: models.DataTypeString},
		{Name: "budget", DataType: models.DataTypeFloat},
		{Name: "team_members", DataType: models.DataTypeJSON},
		{Name: "deadline", DataType: models.DataTypeDate},
		{Name: "owner_id", DataType: models.DataTypeReference},
	}
)

// Test Entity IDs
var (
	EntityID1 = "01HZYX123456789ABCDEFGHJ1"
	EntityID2 = "01HZYX123456789ABCDEFGHJ2"
	EntityID3 = "01HZYX123456789ABCDEFGHJ3"
	EntityID4 = "01HZYX123456789ABCDEFGHJ4"
	EntityID5 = "01HZYX123456789ABCDEFGHJ5"
)

// Test Timestamps
var (
	BaseTime     = time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)
	OneHourLater = BaseTime.Add(time.Hour)
	OneDayLater  = BaseTime.Add(24 * time.Hour)
	OneWeekLater = BaseTime.Add(7 * 24 * time.Hour)
)

// CreateTestContactsTable creates a standard contacts table for testing
func CreateTestContactsTable(userID string) models.TableSchema {
	return models.NewTableSchema(userID, "contacts", ContactsFields)
}

// CreateTestNotesTable creates a standard notes table for testing
func CreateTestNotesTable(userID string) models.TableSchema {
	return models.NewTableSchema(userID, "notes", NotesFields)
}

// CreateTestProjectsTable creates a standard projects table for testing
func CreateTestProjectsTable(userID string) models.TableSchema {
	return models.NewTableSchema(userID, "projects", ProjectsFields)
}

// CreateTestContact creates test contact tuples
func CreateTestContact(userID, entityID string, timestamp time.Time, name, email, phone string, age int) []models.Tuple {
	return []models.Tuple{
		models.NewTuple(userID, entityID, timestamp, "contacts", "name", name),
		models.NewTuple(userID, entityID, timestamp, "contacts", "email", email),
		models.NewTuple(userID, entityID, timestamp, "contacts", "phone", phone),
		models.NewTuple(userID, entityID, timestamp, "contacts", "age", age),
		models.NewTuple(userID, entityID, timestamp, "contacts", "is_active", true),
		models.NewTuple(userID, entityID, timestamp, "contacts", "created_date", timestamp.Format(time.RFC3339)),
		models.NewTuple(userID, entityID, timestamp, "contacts", models.SystemFieldCreated, "true"),
	}
}

// CreateTestNote creates test note tuples
func CreateTestNote(userID, entityID string, timestamp time.Time, title, content string, tags []string) []models.Tuple {
	tagsJSON := `["` + tags[0]
	for _, tag := range tags[1:] {
		tagsJSON += `","` + tag
	}
	tagsJSON += `"]`

	return []models.Tuple{
		models.NewTuple(userID, entityID, timestamp, "notes", "title", title),
		models.NewTuple(userID, entityID, timestamp, "notes", "content", content),
		models.NewTuple(userID, entityID, timestamp, "notes", "tags", tagsJSON),
		models.NewTuple(userID, entityID, timestamp, "notes", "is_public", false),
		models.NewTuple(userID, entityID, timestamp, "notes", "word_count", 150),
		models.NewTuple(userID, entityID, timestamp, "notes", models.SystemFieldCreated, "true"),
	}
}

// CreateUpdateTuples creates tuples that represent updates to existing data
func CreateUpdateTuples(userID, entityID, tableID string, timestamp time.Time, updates map[string]string) []models.Tuple {
	var tuples []models.Tuple
	for fieldName, value := range updates {
		tuples = append(tuples, models.NewTuple(userID, entityID, timestamp, tableID, fieldName, value))
	}
	return tuples
}

// CreateDeleteTuple creates a deletion marker tuple
func CreateDeleteTuple(userID, entityID, tableID string, timestamp time.Time) models.Tuple {
	return models.NewTuple(userID, entityID, timestamp, tableID, models.SystemFieldDeleted, "true")
}

// CreateUndeleteTuple creates an undeletion marker tuple
func CreateUndeleteTuple(userID, entityID, tableID string, timestamp time.Time) models.Tuple {
	return models.NewTuple(userID, entityID, timestamp, tableID, models.SystemFieldDeleted, "false")
}

// TestContactData represents a complete contact for testing
type TestContactData struct {
	EntityID string
	Name     string
	Email    string
	Phone    string
	Age      int
	IsActive bool
}

// SampleContacts provides sample contact data for tests
var SampleContacts = []TestContactData{
	{
		EntityID: EntityID1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Phone:    "+1-555-0101",
		Age:      30,
		IsActive: true,
	},
	{
		EntityID: EntityID2,
		Name:     "Jane Smith",
		Email:    "jane@example.com",
		Phone:    "+1-555-0102",
		Age:      28,
		IsActive: true,
	},
	{
		EntityID: EntityID3,
		Name:     "Bob Johnson",
		Email:    "bob@example.com",
		Phone:    "+1-555-0103",
		Age:      35,
		IsActive: false,
	},
}

// CreateCompleteTestDataset creates a full dataset for comprehensive testing
func CreateCompleteTestDataset(userID string) []models.Tuple {
	var tuples []models.Tuple

	// Create contacts
	for i, contact := range SampleContacts {
		timestamp := BaseTime.Add(time.Duration(i) * time.Hour)
		contactTuples := CreateTestContact(
			userID, contact.EntityID, timestamp,
			contact.Name, contact.Email, contact.Phone, contact.Age,
		)
		tuples = append(tuples, contactTuples...)
	}

	// Add some updates
	updateTime := BaseTime.Add(2 * time.Hour)
	updateTuples := CreateUpdateTuples(userID, EntityID1, "contacts", updateTime, map[string]string{
		"email": "john.doe@newcompany.com",
		"phone": "+1-555-0999",
	})
	tuples = append(tuples, updateTuples...)

	// Delete one contact
	deleteTime := BaseTime.Add(3 * time.Hour)
	deleteTuple := CreateDeleteTuple(userID, EntityID3, "contacts", deleteTime)
	tuples = append(tuples, deleteTuple)

	return tuples
}