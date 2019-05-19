package queries

import (
	"fmt"

	"github.com/javuto/osctrl/pkg/nodes"
	"github.com/jinzhu/gorm"
)

const (
	// QueryTargetPlatform defines platform as target
	QueryTargetPlatform string = "platform"
	// QueryTargetLocalname defines localname as target
	QueryTargetLocalname string = "localname"
	// QueryTargetContext defines context as target
	QueryTargetContext string = "context"
	// QueryTargetUUID defines uuid as target
	QueryTargetUUID string = "uuid"
	// StandardQueryType defines a regular query
	StandardQueryType string = "query"
	// CarveQueryType defines a regular query
	CarveQueryType string = "carve"
	// MetadataQueryType defines a regular query
	MetadataQueryType string = "metadata"
)

// DistributedQuery as abstraction of a distributed query
type DistributedQuery struct {
	gorm.Model
	Name       string `gorm:"not null;unique;index"`
	Creator    string
	Query      string
	Executions int
	Errors     int
	Active     bool
	Completed  bool
	Deleted    bool
	Repeat     uint
	Type       string
	File       string
}

// DistributedQueryTarget to keep target logic for queries
type DistributedQueryTarget struct {
	gorm.Model
	Name  string `gorm:"index"`
	Type  string
	Value string
}

// DistributedQueryExecution to keep track of queries executing
type DistributedQueryExecution struct {
	gorm.Model
	Name   string `gorm:"index"`
	UUID   string `gorm:"index"`
	Result int
}

// QueryReadQueries to hold the on-demand queries
type QueryReadQueries map[string]string

// Queries to handle on-demand queries
type Queries struct {
	DB *gorm.DB
}

// CreateQueries to initialize the queries struct
func CreateQueries(backend *gorm.DB) *Queries {
	var q *Queries
	q = &Queries{DB: backend}
	return q
}

// NodeQueries to get all queries that belong to the provided node
// FIXME this will impact the performance of the TLS endpoint due to being CPU and I/O hungry
// FIMXE potential mitigation can be add a cache (Redis?) layer to store queries per node_key
func (q *Queries) NodeQueries(node nodes.OsqueryNode) (QueryReadQueries, error) {
	// Get all current active queries
	queries, err := q.Gets("active")
	if err != nil {
		return QueryReadQueries{}, err
	}
	// Iterate through active queries, see if they target this node and prepare data in the same loop
	qs := make(QueryReadQueries)
	for _, _q := range queries {
		targets, err := q.GetTargets(_q.Name)
		if err != nil {
			return QueryReadQueries{}, err
		}
		if isQueryTarget(node, targets) && q.NotYetExecuted(_q.Name, node.UUID) {
			qs[_q.Name] = _q.Query
		}
	}
	return qs, nil
}

// Gets all queries by target (active/completed/all/deleted)
func (q *Queries) Gets(target string) ([]DistributedQuery, error) {
	var queries []DistributedQuery
	switch target {
	case "active":
		if err := q.DB.Where("active = ? AND completed = ? AND deleted = ?", true, false, false).Find(&queries).Error; err != nil {
			return queries, err
		}
	case "completed":
		if err := q.DB.Where("active = ? AND completed = ? AND deleted = ?", false, true, false).Find(&queries).Error; err != nil {
			return queries, err
		}
	case "all":
		if err := q.DB.Where("deleted = ?", false).Find(&queries).Error; err != nil {
			return queries, err
		}
	case "deleted":
		if err := q.DB.Where("deleted = ?", true).Find(&queries).Error; err != nil {
			return queries, err
		}
	}
	return queries, nil
}

// Get to get a query by name
func (q *Queries) Get(name string) (DistributedQuery, error) {
	var query DistributedQuery
	if err := q.DB.Where("name = ?", name).Find(&query).Error; err != nil {
		return query, err
	}
	return query, nil
}

// Complete to mark query as completed
func (q *Queries) Complete(name string) error {
	query, err := q.Get(name)
	if err != nil {
		return err
	}
	if err := q.DB.Model(&query).Updates(map[string]interface{}{"completed": true, "active": false}).Error; err != nil {
		return err
	}
	return nil
}

// Activate to mark query as active
func (q *Queries) Activate(name string) error {
	query, err := q.Get(name)
	if err != nil {
		return err
	}
	if err := q.DB.Model(&query).Updates(map[string]interface{}{"completed": false, "active": true}).Error; err != nil {
		return err
	}
	return nil
}

// Delete to mark query as deleted
func (q *Queries) Delete(name string) error {
	query, err := q.Get(name)
	if err != nil {
		return err
	}
	if err := q.DB.Model(&query).Updates(map[string]interface{}{"deleted": true, "active": false}).Error; err != nil {
		return err
	}
	return nil
}

// Create to create new query to be served to nodes
func (q *Queries) Create(query DistributedQuery) error {
	if q.DB.NewRecord(query) {
		if err := q.DB.Create(&query).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("db.NewRecord did not return true")
	}
	return nil
}

// CreateTarget to create target entry for a given query
func (q *Queries) CreateTarget(name, targetType, targetValue string) error {
	queryTarget := DistributedQueryTarget{
		Name:  name,
		Type:  targetType,
		Value: targetValue,
	}
	if q.DB.NewRecord(queryTarget) {
		if err := q.DB.Create(&queryTarget).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("db.NewRecord did not return true")
	}
	return nil
}

// GetTargets to retrieve targets for a given query
func (q *Queries) GetTargets(name string) ([]DistributedQueryTarget, error) {
	var targets []DistributedQueryTarget
	if err := q.DB.Where("name = ?", name).Find(&targets).Error; err != nil {
		return targets, err
	}
	return targets, nil
}

// NotYetExecuted to check if query already executed or it is within the interval
func (q *Queries) NotYetExecuted(name, uuid string) bool {
	var results int
	q.DB.Model(&DistributedQueryExecution{}).Where("name = ? AND uuid = ?", name, uuid).Count(&results)
	return (results == 0)
}

// IncExecution to increase the execution count for this query
func (q *Queries) IncExecution(name string) error {
	query, err := q.Get(name)
	if err != nil {
		return err
	}
	if err := q.DB.Model(&query).Update("executions", query.Executions+1).Error; err != nil {
		return err
	}
	return nil
}

// IncError to increase the error count for this query
func (q *Queries) IncError(name string) error {
	query, err := q.Get(name)
	if err != nil {
		return err
	}
	if err := q.DB.Model(&query).Update("errors", query.Errors+1).Error; err != nil {
		return err
	}
	return nil
}

// TrackExecution to keep track of where queries have already ran
func (q *Queries) TrackExecution(name, uuid string, result int) error {
	queryExecution := DistributedQueryExecution{
		Name:   name,
		UUID:   uuid,
		Result: result,
	}
	if q.DB.NewRecord(queryExecution) {
		if err := q.DB.Create(&queryExecution).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("db.NewRecord did not return true")
	}
	return nil
}
