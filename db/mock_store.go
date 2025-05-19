package db

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MockStore implements the Store interface for testing
type MockStore struct {
	mu            sync.RWMutex
	facts         map[string]Fact // Key is "userID#timestamp#id"
	tableCreated  bool
	tableDeleted  bool
	failureMode   map[string]error // Map of operation names to errors for testing failure scenarios
	callRecords   map[string]int   // Tracks number of calls to each method
	expectedCalls map[string]int   // Expected number of calls for verification
}

// NewMockStore creates a new mock store for testing
func NewMockStore() *MockStore {
	return &MockStore{
		facts:         make(map[string]Fact),
		failureMode:   make(map[string]error),
		callRecords:   make(map[string]int),
		expectedCalls: make(map[string]int),
	}
}

// SimulateFailure sets an error to be returned when the specified operation is called
func (s *MockStore) SimulateFailure(operation string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureMode[operation] = err
}

// ExpectCall sets the expected number of calls for a method
func (s *MockStore) ExpectCall(operation string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expectedCalls[operation] = count
}

// VerifyExpectations checks if all expected calls were made
func (s *MockStore) VerifyExpectations() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for op, expected := range s.expectedCalls {
		actual := s.callRecords[op]
		if actual != expected {
			return fmt.Errorf("expected %d calls to %s, got %d", expected, op, actual)
		}
	}
	return nil
}

// recordCall increments the call counter for an operation
func (s *MockStore) recordCall(operation string) {
	s.callRecords[operation]++
}

// checkFailure returns the configured error for an operation if set
func (s *MockStore) checkFailure(operation string) error {
	if err, exists := s.failureMode[operation]; exists {
		return &StoreError{
			Operation: operation,
			Err:       err,
		}
	}
	return nil
}

// CreateTable implements Store.CreateTable
func (s *MockStore) CreateTable(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordCall("CreateTable")
	
	if err := s.checkFailure("CreateTable"); err != nil {
		return err
	}
	
	s.tableCreated = true
	return nil
}

// DeleteTable implements Store.DeleteTable
func (s *MockStore) DeleteTable(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordCall("DeleteTable")
	
	if err := s.checkFailure("DeleteTable"); err != nil {
		return err
	}
	
	s.tableCreated = false
	s.tableDeleted = true
	s.facts = make(map[string]Fact) // Clear all data
	return nil
}

// PutFact implements Store.PutFact
func (s *MockStore) PutFact(ctx context.Context, fact *Fact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordCall("PutFact")
	
	if err := s.checkFailure("PutFact"); err != nil {
		return err
	}
	
	if fact == nil {
		return &StoreError{
			Operation: "PutFact",
			Err:       fmt.Errorf("fact cannot be nil"),
		}
	}
	
	if fact.ID == "" {
		return &StoreError{
			Operation: "PutFact",
			Err:       fmt.Errorf("fact ID cannot be empty"),
		}
	}
	
	if !s.tableCreated {
		return &StoreError{
			Operation: "PutFact",
			Err:       fmt.Errorf("table not created"),
		}
	}
	
	// Create a deep copy to avoid external modification
	factCopy := *fact
	key := fmt.Sprintf("%s#%s#%s", fact.UserID, fact.Timestamp.Format(time.RFC3339Nano), fact.ID)
	s.facts[key] = factCopy
	
	return nil
}

// GetFact implements Store.GetFact
func (s *MockStore) GetFact(ctx context.Context, id string) (*Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.recordCall("GetFact")
	
	if err := s.checkFailure("GetFact"); err != nil {
		return nil, err
	}
	
	if !s.tableCreated {
		return nil, &StoreError{
			Operation: "GetFact",
			Err:       fmt.Errorf("table not created"),
		}
	}
	
	var latestFact *Fact
	var latestTime time.Time
	
	// Find the most recent version of this fact
	for _, fact := range s.facts {
		if fact.ID == id && fact.Timestamp.After(latestTime) {
			factCopy := fact
			latestFact = &factCopy
			latestTime = fact.Timestamp
		}
	}
	
	if latestFact == nil {
		return nil, &StoreError{
			Operation: "GetFact",
			Err:       fmt.Errorf("fact not found"),
		}
	}
	
	return latestFact, nil
}

// DeleteFact implements Store.DeleteFact
func (s *MockStore) DeleteFact(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordCall("DeleteFact")
	
	if err := s.checkFailure("DeleteFact"); err != nil {
		return err
	}
	
	if !s.tableCreated {
		return &StoreError{
			Operation: "DeleteFact",
			Err:       fmt.Errorf("table not created"),
		}
	}
	
	// Find the fact first
	var foundFact *Fact
	var latestTime time.Time
	
	for _, fact := range s.facts {
		if fact.ID == id && fact.Timestamp.After(latestTime) {
			factCopy := fact
			foundFact = &factCopy
			latestTime = fact.Timestamp
		}
	}
	
	if foundFact == nil {
		return &StoreError{
			Operation: "DeleteFact",
			Err:       fmt.Errorf("fact not found"),
		}
	}
	
	// Create a deletion marker
	deletedFact := *foundFact
	deletedFact.IsDeleted = true
	deletedFact.Timestamp = time.Now()
	
	key := fmt.Sprintf("%s#%s#%s", deletedFact.UserID, deletedFact.Timestamp.Format(time.RFC3339Nano), deletedFact.ID)
	s.facts[key] = deletedFact
	
	return nil
}

// QueryByField implements Store.QueryByField
func (s *MockStore) QueryByField(ctx context.Context, namespace, fieldName string, opts QueryOptions) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.recordCall("QueryByField")
	
	if err := s.checkFailure("QueryByField"); err != nil {
		return nil, err
	}
	
	if !s.tableCreated {
		return nil, &StoreError{
			Operation: "QueryByField",
			Err:       fmt.Errorf("table not created"),
		}
	}
	
	var results []Fact
	
	// Filter by namespace and fieldName
	for _, fact := range s.facts {
		if fact.Namespace == namespace && fact.FieldName == fieldName {
			// Apply time range filter if provided
			if opts.StartTime != nil && opts.EndTime != nil {
				if fact.Timestamp.Before(*opts.StartTime) || fact.Timestamp.After(*opts.EndTime) {
					continue
				}
			}
			results = append(results, fact)
		}
	}
	
	// Sort by timestamp
	sort.Slice(results, func(i, j int) bool {
		if opts.SortAscending {
			return results[i].Timestamp.Before(results[j].Timestamp)
		}
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	
	// Apply limit if provided
	if opts.Limit != nil && int(*opts.Limit) < len(results) {
		results = results[:*opts.Limit]
	}
	
	// No pagination in mock implementation
	return &QueryResult{
		Facts:     results,
		NextToken: nil,
	}, nil
}

// QueryByTimeRange implements Store.QueryByTimeRange
func (s *MockStore) QueryByTimeRange(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.recordCall("QueryByTimeRange")
	
	if err := s.checkFailure("QueryByTimeRange"); err != nil {
		return nil, err
	}
	
	if !s.tableCreated {
		return nil, &StoreError{
			Operation: "QueryByTimeRange",
			Err:       fmt.Errorf("table not created"),
		}
	}
	
	var results []Fact
	
	// Filter by time range
	for _, fact := range s.facts {
		if opts.StartTime != nil && opts.EndTime != nil {
			if fact.Timestamp.Before(*opts.StartTime) || fact.Timestamp.After(*opts.EndTime) {
				continue
			}
		}
		results = append(results, fact)
	}
	
	// Sort by timestamp
	sort.Slice(results, func(i, j int) bool {
		if opts.SortAscending {
			return results[i].Timestamp.Before(results[j].Timestamp)
		}
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	
	// Apply limit if provided
	if opts.Limit != nil && int(*opts.Limit) < len(results) {
		results = results[:*opts.Limit]
	}
	
	// No pagination in mock implementation
	return &QueryResult{
		Facts:     results,
		NextToken: nil,
	}, nil
}

// QueryByNamespace implements Store.QueryByNamespace
func (s *MockStore) QueryByNamespace(ctx context.Context, namespace string, opts QueryOptions) (*QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.recordCall("QueryByNamespace")
	
	if err := s.checkFailure("QueryByNamespace"); err != nil {
		return nil, err
	}
	
	if !s.tableCreated {
		return nil, &StoreError{
			Operation: "QueryByNamespace",
			Err:       fmt.Errorf("table not created"),
		}
	}
	
	var results []Fact
	
	// Filter by namespace
	for _, fact := range s.facts {
		if fact.Namespace == namespace {
			// Apply time range filter if provided
			if opts.StartTime != nil && opts.EndTime != nil {
				if fact.Timestamp.Before(*opts.StartTime) || fact.Timestamp.After(*opts.EndTime) {
					continue
				}
			}
			results = append(results, fact)
		}
	}
	
	// Sort by timestamp
	sort.Slice(results, func(i, j int) bool {
		if opts.SortAscending {
			return results[i].Timestamp.Before(results[j].Timestamp)
		}
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	
	// Apply limit if provided
	if opts.Limit != nil && int(*opts.Limit) < len(results) {
		results = results[:*opts.Limit]
	}
	
	// No pagination in mock implementation
	return &QueryResult{
		Facts:     results,
		NextToken: nil,
	}, nil
}

// GetSnapshotAtTime implements Store.GetSnapshotAtTime
func (s *MockStore) GetSnapshotAtTime(ctx context.Context, namespace string, at time.Time) (map[string]Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.recordCall("GetSnapshotAtTime")
	
	if err := s.checkFailure("GetSnapshotAtTime"); err != nil {
		return nil, err
	}
	
	if !s.tableCreated {
		return nil, &StoreError{
			Operation: "GetSnapshotAtTime",
			Err:       fmt.Errorf("table not created"),
		}
	}
	
	// Filter facts by namespace and timestamp <= at
	relevantFacts := make([]Fact, 0)
	for _, fact := range s.facts {
		if (namespace == "" || fact.Namespace == namespace) && !fact.Timestamp.After(at) {
			relevantFacts = append(relevantFacts, fact)
		}
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(relevantFacts, func(i, j int) bool {
		return relevantFacts[i].Timestamp.After(relevantFacts[j].Timestamp)
	})
	
	// Group facts by field key
	fieldFactMap := make(map[string][]Fact)
	for _, fact := range relevantFacts {
		key := fmt.Sprintf("%s#%s", fact.Namespace, fact.FieldName)
		fieldFactMap[key] = append(fieldFactMap[key], fact)
	}
	
	// Build snapshot with latest version of each field
	snapshot := make(map[string]Fact)
	for key, facts := range fieldFactMap {
		// Facts are already sorted newest first
		if len(facts) > 0 && !facts[0].IsDeleted {
			snapshot[key] = facts[0]
		}
	}
	
	return snapshot, nil
}