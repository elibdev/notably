package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elibdev/notably/dynamo"
)

// StoreAdapter adapts our new Store interface to work with the existing API
// This allows for a gradual migration from the old dynamo.Client to the new Store interface
type StoreAdapter struct {
	store Store
}

// NewStoreAdapter creates a new adapter around a Store implementation
func NewStoreAdapter(store Store) *StoreAdapter {
	return &StoreAdapter{
		store: store,
	}
}

// CreateTable implements the same functionality as dynamo.Client.CreateTable
func (a *StoreAdapter) CreateTable(ctx context.Context) error {
	return a.store.CreateTable(ctx)
}

// PutFact adapts between the dynamo.Fact type and our db.Fact type
func (a *StoreAdapter) PutFact(ctx context.Context, fact dynamo.Fact) error {
	dbFact := convertFromLegacyFact(fact)
	return a.store.PutFact(ctx, &dbFact)
}

// QueryByField performs a field query using our new Store interface
func (a *StoreAdapter) QueryByField(ctx context.Context, namespace, fieldName string, start, end time.Time) ([]dynamo.Fact, error) {
	opts := QueryOptions{
		StartTime:     &start,
		EndTime:       &end,
		SortAscending: true,
	}

	result, err := a.store.QueryByField(ctx, namespace, fieldName, opts)
	if err != nil {
		return nil, err
	}

	return convertToLegacyFacts(result.Facts), nil
}

// QueryByTimeRange performs a time range query using our new Store interface
func (a *StoreAdapter) QueryByTimeRange(ctx context.Context, start, end time.Time) ([]dynamo.Fact, error) {
	opts := QueryOptions{
		StartTime:     &start,
		EndTime:       &end,
		SortAscending: true,
	}

	result, err := a.store.QueryByTimeRange(ctx, opts)
	if err != nil {
		return nil, err
	}

	return convertToLegacyFacts(result.Facts), nil
}

// GetFactByID retrieves a single fact by ID (not in the original interface but useful)
func (a *StoreAdapter) GetFactByID(ctx context.Context, id string) (*dynamo.Fact, error) {
	fact, err := a.store.GetFact(ctx, id)
	if err != nil {
		return nil, err
	}

	legacyFact := convertToLegacyFact(*fact)
	return &legacyFact, nil
}

// DeleteFactByID performs a soft delete of a fact
func (a *StoreAdapter) DeleteFactByID(ctx context.Context, id string) error {
	return a.store.DeleteFact(ctx, id)
}

// GetSnapshot retrieves a snapshot of all facts at a given time
func (a *StoreAdapter) GetSnapshot(ctx context.Context, at time.Time) (map[string]map[string]dynamo.Fact, error) {
	// First get a snapshot with our new interface
	allNamespaces, err := a.store.GetSnapshotAtTime(ctx, "", at)
	if err != nil {
		return nil, err
	}

	// Group facts by namespace and field name
	result := make(map[string]map[string]dynamo.Fact)

	for _, fact := range allNamespaces {
		ns := fact.Namespace
		if _, exists := result[ns]; !exists {
			result[ns] = make(map[string]dynamo.Fact)
		}

		result[ns][fact.FieldName] = convertToLegacyFact(fact)
	}

	return result, nil
}

// convertFromLegacyFact converts a dynamo.Fact to a db.Fact
func convertFromLegacyFact(legacy dynamo.Fact) Fact {
	// Convert value to string safely
	var valueStr string
	if legacy.Value != nil {
		// For JSON data types, marshal to proper JSON
		if legacy.DataType == "json" {
			jsonBytes, err := json.Marshal(legacy.Value)
			if err != nil {
				// Fall back to string representation if marshaling fails
				valueStr = fmt.Sprintf("%v", legacy.Value)
			} else {
				valueStr = string(jsonBytes)
			}
		} else {
			valueStr = fmt.Sprintf("%v", legacy.Value)
		}
	}

	// Convert columns
	var columns []ColumnDefinition
	if len(legacy.Columns) > 0 {
		columns = make([]ColumnDefinition, len(legacy.Columns))
		for i, col := range legacy.Columns {
			columns[i] = ColumnDefinition{
				Name:     col.Name,
				DataType: col.DataType,
			}
		}
	}

	return Fact{
		ID:        legacy.ID,
		Timestamp: legacy.Timestamp,
		Namespace: legacy.Namespace,
		FieldName: legacy.FieldName,
		DataType:  DataType(legacy.DataType),
		Value:     valueStr,
		Columns:   columns,
		IsDeleted: legacy.DataType == "deleted",
	}
}

// convertToLegacyFact converts a db.Fact to a dynamo.Fact
func convertToLegacyFact(fact Fact) dynamo.Fact {
	// Convert columns
	var columns []dynamo.ColumnDefinition
	if len(fact.Columns) > 0 {
		columns = make([]dynamo.ColumnDefinition, len(fact.Columns))
		for i, col := range fact.Columns {
			columns[i] = dynamo.ColumnDefinition{
				Name:     col.Name,
				DataType: col.DataType,
			}
		}
	}

	// Convert value back to proper type for JSON data
	var value interface{}
	if string(fact.DataType) == "json" && fact.Value != "" {
		// Parse JSON string back to interface{}
		err := json.Unmarshal([]byte(fact.Value), &value)
		if err != nil {
			// Fall back to string value if parsing fails
			value = fact.Value
		}
	} else {
		value = fact.Value
	}

	return dynamo.Fact{
		ID:        fact.ID,
		Timestamp: fact.Timestamp,
		Namespace: fact.Namespace,
		FieldName: fact.FieldName,
		DataType:  string(fact.DataType),
		Value:     value,
		Columns:   columns,
	}
}

// convertToLegacyFacts converts a slice of db.Fact to a slice of dynamo.Fact
func convertToLegacyFacts(facts []Fact) []dynamo.Fact {
	result := make([]dynamo.Fact, len(facts))
	for i, fact := range facts {
		result[i] = convertToLegacyFact(fact)
	}
	return result
}

// CreateStoreFromClient creates a new Store implementation wrapping the existing dynamo.Client
// This allows for gradually adopting the new interfaces with existing client code
func CreateStoreFromClient(client *dynamo.Client) Store {
	// Implementation depends on internal details of dynamo.Client
	// This is a simplified adapter and would need more work in a real implementation
	return &LegacyClientAdapter{
		client: client,
	}
}

// LegacyClientAdapter adapts the existing dynamo.Client to our new Store interface
type LegacyClientAdapter struct {
	client *dynamo.Client
}

// Implement the Store interface methods using the legacy client
func (a *LegacyClientAdapter) CreateTable(ctx context.Context) error {
	return a.client.CreateTable(ctx)
}

func (a *LegacyClientAdapter) DeleteTable(ctx context.Context) error {
	// Not supported in the original client
	return &StoreError{
		Operation: "DeleteTable",
		Err:       ErrNotImplemented,
	}
}

func (a *LegacyClientAdapter) PutFact(ctx context.Context, fact *Fact) error {
	if fact == nil {
		return &StoreError{
			Operation: "PutFact",
			Err:       fmt.Errorf("fact cannot be nil"),
		}
	}

	// Convert to legacy fact type
	legacyFact := convertToLegacyFact(*fact)

	return a.client.PutFact(ctx, legacyFact)
}

func (a *LegacyClientAdapter) GetFact(ctx context.Context, id string) (*Fact, error) {
	// Legacy client doesn't have a direct GetFact method
	// We'll need to query for it and find the latest version

	// Use a large time range to find the fact
	endTime := time.Now().UTC()
	startTime := time.Unix(0, 0) // Beginning of time

	// Query all facts
	facts, err := a.client.QueryByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return nil, &StoreError{
			Operation: "GetFact",
			Err:       err,
		}
	}

	// Find the most recent fact with the given ID
	var latestFact *dynamo.Fact
	for i, f := range facts {
		if f.ID == id {
			if latestFact == nil || f.Timestamp.After(latestFact.Timestamp) {
				factCopy := facts[i]
				latestFact = &factCopy
			}
		}
	}

	if latestFact == nil {
		return nil, &StoreError{
			Operation: "GetFact",
			Err:       fmt.Errorf("fact not found"),
		}
	}

	// Convert to our Fact type
	result := convertFromLegacyFact(*latestFact)

	return &result, nil
}

func (a *LegacyClientAdapter) DeleteFact(ctx context.Context, id string) error {
	// First get the latest version of the fact
	fact, err := a.GetFact(ctx, id)
	if err != nil {
		return &StoreError{
			Operation: "DeleteFact",
			Err:       err,
		}
	}

	// Create a deletion marker in DynamoDB
	legacyFact := dynamo.Fact{
		ID:        id,
		Timestamp: time.Now().UTC(),
		Namespace: fact.Namespace,
		FieldName: fact.FieldName,
		DataType:  "deleted",
		Value:     nil,
	}

	return a.client.PutFact(ctx, legacyFact)
}

func (a *LegacyClientAdapter) QueryByField(ctx context.Context, namespace, fieldName string, opts QueryOptions) (*QueryResult, error) {
	// Set default start/end times if not provided
	startTime := time.Unix(0, 0)
	if opts.StartTime != nil {
		startTime = *opts.StartTime
	}

	endTime := time.Now().UTC()
	if opts.EndTime != nil {
		endTime = *opts.EndTime
	}

	// Call the legacy client method
	facts, err := a.client.QueryByField(ctx, namespace, fieldName, startTime, endTime)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByField",
			Err:       err,
		}
	}

	// Convert to our Fact type
	result := make([]Fact, len(facts))
	for i, f := range facts {
		result[i] = convertFromLegacyFact(f)
	}

	// Sort if needed
	if opts.SortAscending {
		// Facts should already be sorted ascending by the DynamoDB query
	} else {
		// Reverse the slice
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	return &QueryResult{
		Facts:     result,
		NextToken: nil, // Legacy client doesn't support pagination
	}, nil
}

func (a *LegacyClientAdapter) QueryByTimeRange(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	// Set default start/end times if not provided
	startTime := time.Unix(0, 0)
	if opts.StartTime != nil {
		startTime = *opts.StartTime
	}

	endTime := time.Now().UTC()
	if opts.EndTime != nil {
		endTime = *opts.EndTime
	}

	// Call the legacy client method
	facts, err := a.client.QueryByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByTimeRange",
			Err:       err,
		}
	}

	// Convert to our Fact type
	result := make([]Fact, len(facts))
	for i, f := range facts {
		result[i] = convertFromLegacyFact(f)
	}

	// Sort if needed
	if opts.SortAscending {
		// Facts should already be sorted ascending by the DynamoDB query
	} else {
		// Reverse the slice
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	return &QueryResult{
		Facts:     result,
		NextToken: nil, // Legacy client doesn't support pagination
	}, nil
}

func (a *LegacyClientAdapter) QueryByNamespace(ctx context.Context, namespace string, opts QueryOptions) (*QueryResult, error) {
	// Legacy client doesn't have this method directly
	// We'll need to get all facts and filter by namespace

	// First get all facts in the time range
	result, err := a.QueryByTimeRange(ctx, opts)
	if err != nil {
		return nil, &StoreError{
			Operation: "QueryByNamespace",
			Err:       err,
		}
	}

	// Filter by namespace
	filteredFacts := make([]Fact, 0)
	for _, fact := range result.Facts {
		if fact.Namespace == namespace {
			filteredFacts = append(filteredFacts, fact)
		}
	}

	return &QueryResult{
		Facts:     filteredFacts,
		NextToken: nil,
	}, nil
}

func (a *LegacyClientAdapter) GetSnapshotAtTime(ctx context.Context, namespace string, at time.Time) (map[string]Fact, error) {
	// Get all facts up to the time "at"
	startTime := time.Unix(0, 0)
	endTime := at

	opts := QueryOptions{
		StartTime:     &startTime,
		EndTime:       &endTime,
		SortAscending: false, // Most recent first
	}

	var result *QueryResult
	var err error

	if namespace == "" {
		// Get all facts
		result, err = a.QueryByTimeRange(ctx, opts)
	} else {
		// Filter by namespace
		result, err = a.QueryByNamespace(ctx, namespace, opts)
	}

	if err != nil {
		return nil, &StoreError{
			Operation: "GetSnapshotAtTime",
			Err:       err,
		}
	}

	// Create snapshot map (using namespace#fieldName as key)
	snapshot := make(map[string]Fact)
	for _, fact := range result.Facts {
		key := fmt.Sprintf("%s#%s", fact.Namespace, fact.FieldName)

		// If we haven't seen this field yet or this is a newer version
		if existing, exists := snapshot[key]; !exists || fact.Timestamp.After(existing.Timestamp) {
			if !fact.IsDeleted {
				snapshot[key] = fact
			} else if exists {
				// If it's a deletion marker and we had this field, remove it
				delete(snapshot, key)
			}
		}
	}

	return snapshot, nil
}

// ErrNotImplemented is returned for operations not supported by the legacy client
var ErrNotImplemented = fmt.Errorf("operation not implemented in legacy client")
