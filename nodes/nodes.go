package nodes

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// OsqueryNode as abstraction of a node
type OsqueryNode struct {
	gorm.Model
	NodeKey         string `gorm:"index"`
	UUID            string `gorm:"index"`
	Platform        string
	PlatformVersion string
	OsqueryVersion  string
	Hostname        string
	Localname       string
	IPAddress       string
	Username        string
	OsqueryUser     string
	Environment     string
	CPU             string
	Memory          string
	HardwareSerial  string
	DaemonHash      string
	ConfigHash      string
	RawEnrollment   json.RawMessage `gorm:"type:text"`
	LastStatus      time.Time
	LastResult      time.Time
	LastConfig      time.Time
	LastQueryRead   time.Time
	LastQueryWrite  time.Time
}

// ArchiveOsqueryNode as abstraction of an archived node
type ArchiveOsqueryNode struct {
	gorm.Model
	NodeKey         string `gorm:"index"`
	UUID            string `gorm:"index"`
	Trigger         string
	Platform        string
	PlatformVersion string
	OsqueryVersion  string
	Hostname        string
	Localname       string
	IPAddress       string
	Username        string
	OsqueryUser     string
	Environment     string
	CPU             string
	Memory          string
	HardwareSerial  string
	ConfigHash      string
	DaemonHash      string
	RawEnrollment   json.RawMessage
	LastStatus      time.Time
	LastResult      time.Time
	LastConfig      time.Time
	LastQueryRead   time.Time
	LastQueryWrite  time.Time
}

// NodeHistoryIPAddress to keep track of all IP Addresses for nodes
type NodeHistoryIPAddress struct {
	gorm.Model
	UUID      string `gorm:"index"`
	IPAddress string
	Count     int
}

// NodeHistoryHostname to keep track of all IP Addresses for nodes
type NodeHistoryHostname struct {
	gorm.Model
	UUID     string `gorm:"index"`
	Hostname string
	Count    int
}

// NodeHistoryLocalname to keep track of all IP Addresses for nodes
type NodeHistoryLocalname struct {
	gorm.Model
	UUID      string `gorm:"index"`
	Localname string
	Count     int
}

// NodeHistoryUsername to keep track of all usernames for nodes
type NodeHistoryUsername struct {
	gorm.Model
	UUID     string `gorm:"index"`
	Username string
	Count    int
}

// StatsData to display node stats
type StatsData struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
}

// NodeManager to handle all nodes of the system
type NodeManager struct {
	DB *gorm.DB
}

// CreateNodes to initialize the nodes struct and its tables
func CreateNodes(backend *gorm.DB) *NodeManager {
	var n *NodeManager
	n = &NodeManager{DB: backend}
	// table osquery_nodes
	if err := backend.AutoMigrate(OsqueryNode{}).Error; err != nil {
		log.Fatalf("Failed to AutoMigrate table (osquery_nodes): %v", err)
	}
	// table archive_osquery_nodes
	if err := backend.AutoMigrate(ArchiveOsqueryNode{}).Error; err != nil {
		log.Fatalf("Failed to AutoMigrate table (archive_osquery_nodes): %v", err)
	}
	// table node_history_ipaddress
	if err := backend.AutoMigrate(NodeHistoryIPAddress{}).Error; err != nil {
		log.Fatalf("Failed to AutoMigrate table (node_history_ipaddress): %v", err)
	}
	// table node_history_hostname
	if err := backend.AutoMigrate(NodeHistoryHostname{}).Error; err != nil {
		log.Fatalf("Failed to AutoMigrate table (node_history_hostname): %v", err)
	}
	// table node_history_localname
	if err := backend.AutoMigrate(NodeHistoryLocalname{}).Error; err != nil {
		log.Fatalf("Failed to AutoMigrate table (node_history_localname): %v", err)
	}
	// table node_history_username
	if err := backend.AutoMigrate(NodeHistoryUsername{}).Error; err != nil {
		log.Fatalf("Failed to AutoMigrate table (node_history_username): %v", err)
	}
	return n
}

// CheckByKey to check if node exists by node_key
// node_key is expected lowercase
func (n *NodeManager) CheckByKey(nodeKey string) bool {
	var results int
	n.DB.Model(&OsqueryNode{}).Where("node_key = ?", strings.ToLower(nodeKey)).Count(&results)
	return (results > 0)
}

// CheckByUUID to check if node exists by UUID
// UUID is expected uppercase
func (n *NodeManager) CheckByUUID(uuid string) bool {
	var results int
	n.DB.Model(&OsqueryNode{}).Where("uuid = ?", strings.ToUpper(uuid)).Count(&results)
	return (results > 0)
}

// CheckByUUIDEnv to check if node exists by UUID in a specific environment
// UUID is expected uppercase
func (n *NodeManager) CheckByUUIDEnv(uuid, environment string) bool {
	var results int
	n.DB.Model(&OsqueryNode{}).Where("uuid = ? AND environment = ?", strings.ToUpper(uuid), environment).Count(&results)
	return (results > 0)
}

// CheckByHost to check if node exists by Hostname
func (n *NodeManager) CheckByHost(host string) bool {
	var results int
	n.DB.Model(&OsqueryNode{}).Where("hostname = ? OR localname = ?", host, host).Count(&results)
	return (results > 0)
}

// GetByKey to retrieve full node object from DB, by node_key
// node_key is expected lowercase
func (n *NodeManager) GetByKey(nodekey string) (OsqueryNode, error) {
	var node OsqueryNode
	if err := n.DB.Where("node_key = ?", strings.ToLower(nodekey)).First(&node).Error; err != nil {
		return node, err
	}
	return node, nil
}

// GetByUUID to retrieve full node object from DB, by uuid
// UUID is expected uppercase
func (n *NodeManager) GetByUUID(uuid string) (OsqueryNode, error) {
	var node OsqueryNode
	if err := n.DB.Where("uuid = ?", strings.ToUpper(uuid)).First(&node).Error; err != nil {
		return node, err
	}
	return node, nil
}

// GetBySelector to retrieve target nodes by selector
func (n *NodeManager) GetBySelector(stype, selector, target string, hours int64) ([]OsqueryNode, error) {
	var nodes []OsqueryNode
	var s string
	switch stype {
	case "environment":
		s = "environment"
	case "platform":
		s = "platform"
	}
	switch target {
	case "all":
		if err := n.DB.Where(s+" = ?", selector).Find(&nodes).Error; err != nil {
			return nodes, err
		}
	case "active":
		//if err := n.DB.Where(s+" = ?", selector).Where("updated_at > ?", time.Now().AddDate(0, 0, -3)).Find(&nodes).Error; err != nil {
		if err := n.DB.Where(s+" = ?", selector).Where("updated_at > ?", time.Now().Add(time.Duration(hours)*time.Hour)).Find(&nodes).Error; err != nil {
			return nodes, err
		}
	case "inactive":
		//if err := n.DB.Where(s+" = ?", selector).Where("updated_at < ?", time.Now().AddDate(0, 0, -3)).Find(&nodes).Error; err != nil {
		if err := n.DB.Where(s+" = ?", selector).Where("updated_at < ?", time.Now().Add(time.Duration(hours)*time.Hour)).Find(&nodes).Error; err != nil {
			return nodes, err
		}
	}
	return nodes, nil
}

// Gets to retrieve all/active/inactive nodes
func (n *NodeManager) Gets(target string, hours int64) ([]OsqueryNode, error) {
	var nodes []OsqueryNode
	switch target {
	case "all":
		if err := n.DB.Find(&nodes).Error; err != nil {
			return nodes, err
		}
	case "active":
		//if err := n.DB.Where("updated_at > ?", time.Now().AddDate(0, 0, -3)).Find(&nodes).Error; err != nil {
		if err := n.DB.Where("updated_at > ?", time.Now().Add(time.Duration(hours)*time.Hour)).Find(&nodes).Error; err != nil {
			return nodes, err
		}
	case "inactive":
		//if err := n.DB.Where("updated_at < ?", time.Now().AddDate(0, 0, -3)).Find(&nodes).Error; err != nil {
		if err := n.DB.Where("updated_at < ?", time.Now().Add(time.Duration(hours)*time.Hour)).Find(&nodes).Error; err != nil {
			return nodes, err
		}
	}
	return nodes, nil
}

// GetByEnv to retrieve target nodes by environment
func (n *NodeManager) GetByEnv(environment, target string, hours int64) ([]OsqueryNode, error) {
	return n.GetBySelector("environment", environment, target, hours)
}

// GetByPlatform to retrieve target nodes by platform
func (n *NodeManager) GetByPlatform(platform, target string, hours int64) ([]OsqueryNode, error) {
	return n.GetBySelector("platform", platform, target, hours)
}

// GetAllPlatforms to get all different platform with nodes in them
func (n *NodeManager) GetAllPlatforms() ([]string, error) {
	var platforms []string
	var platform string
	rows, err := n.DB.Table("osquery_nodes").Select("DISTINCT(platform)").Rows()
	if err != nil {
		return platforms, nil
	}
	for rows.Next() {
		_ = rows.Scan(&platform)
		platforms = append(platforms, platform)
	}
	return platforms, nil
}

// GetStatsByEnv to populate table stats about nodes by environment. Active machine is < 3 days
func (n *NodeManager) GetStatsByEnv(environment string, hours int64) (StatsData, error) {
	var stats StatsData
	if err := n.DB.Model(&OsqueryNode{}).Where("environment = ?", environment).Count(&stats.Total).Error; err != nil {
		return stats, err
	}
	tHours := time.Now().Add(time.Duration(hours) * time.Hour)
	if err := n.DB.Model(&OsqueryNode{}).Where("environment = ?", environment).Where("updated_at > ?", tHours).Count(&stats.Active).Error; err != nil {
		return stats, err
	}
	if err := n.DB.Model(&OsqueryNode{}).Where("environment = ?", environment).Where("updated_at < ?", tHours).Count(&stats.Inactive).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

// GetStatsByPlatform to populate table stats about nodes by platform. Active machine is < 3 days
func (n *NodeManager) GetStatsByPlatform(platform string, hours int64) (StatsData, error) {
	var stats StatsData
	if err := n.DB.Model(&OsqueryNode{}).Where("platform = ?", platform).Count(&stats.Total).Error; err != nil {
		return stats, err
	}
	tHours := time.Now().Add(time.Duration(hours) * time.Hour)
	if err := n.DB.Model(&OsqueryNode{}).Where("platform = ?", platform).Where("updated_at > ?", tHours).Count(&stats.Active).Error; err != nil {
		return stats, err
	}
	if err := n.DB.Model(&OsqueryNode{}).Where("platform = ?", platform).Where("updated_at < ?", tHours).Count(&stats.Inactive).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

// UpdateMetadataByUUID to update node metadata by UUID
func (n *NodeManager) UpdateMetadataByUUID(uuid string, metadata NodeMetadata) error {
	// Retrieve node
	node, err := n.GetByUUID(uuid)
	if err != nil {
		return fmt.Errorf("getNodeByUUID %v", err)
	}
	// Prepare data
	data := OsqueryNode{
		OsqueryUser:    "",
		Username:       "",
		Hostname:       "",
		Localname:      "",
		IPAddress:      "",
		ConfigHash:     "",
		DaemonHash:     "",
		OsqueryVersion: "",
	}
	// System user metadata update, if different
	if (metadata.Username != "") && (metadata.Username != node.Username) {
		data.Username = metadata.Username
		e := NodeHistoryUsername{
			UUID:     node.UUID,
			Username: metadata.Username,
		}
		if err := n.NewHistoryUsername(e); err != nil {
			return fmt.Errorf("newNodeHistoryUsername %v", err)
		}
	}
	// Osquery user metadata update, if different
	if (metadata.OsqueryUser != "") && (metadata.OsqueryUser != node.OsqueryUser) {
		data.OsqueryUser = metadata.OsqueryUser
	}
	// Hostname metadata update, if different
	if (metadata.Hostname != "") && (metadata.Hostname != node.Hostname) {
		data.Hostname = metadata.Hostname
		e := NodeHistoryHostname{
			UUID:     node.UUID,
			Hostname: metadata.Hostname,
		}
		if err := n.NewHistoryHostname(e); err != nil {
			return fmt.Errorf("newNodeHistoryHostname %v", err)
		}
	}
	// Localname metadata update, if different
	if (metadata.Localname != "") && (metadata.Localname != node.Localname) {
		data.Localname = metadata.Localname
		e := NodeHistoryLocalname{
			UUID:      node.UUID,
			Localname: metadata.Localname,
		}
		if err := n.NewHistoryLocalname(e); err != nil {
			return fmt.Errorf("newNodeHistoryLocalname %v", err)
		}
	}
	// IP Address metadata update, if different
	if (metadata.IPAddress != "") && (metadata.IPAddress != node.IPAddress) {
		data.IPAddress = metadata.IPAddress
		e := NodeHistoryIPAddress{
			UUID:      node.UUID,
			IPAddress: metadata.IPAddress,
			Count:     1,
		}
		if err := n.NewHistoryIPAddress(e); err != nil {
			return fmt.Errorf("newNodeHistoryIPAddress %v", err)
		}
	} else if err := n.IncHistoryIPAddress(node.UUID, metadata.IPAddress); err != nil {
		return fmt.Errorf("incNodeHistoryIPAddress %v", err)
	}
	// Osquery configuration metadata update, if different
	if (metadata.ConfigHash != "") && (metadata.ConfigHash != node.ConfigHash) {
		data.ConfigHash = metadata.ConfigHash
	}
	// Osquery daemon hash update, if different
	if (metadata.DaemonHash != "") && (metadata.DaemonHash != node.DaemonHash) {
		data.DaemonHash = metadata.DaemonHash
	}
	// Osquery version metadata update, if different
	if (metadata.OsqueryVersion != "") && (metadata.OsqueryVersion != node.OsqueryVersion) {
		data.OsqueryVersion = metadata.OsqueryVersion
	}
	if err := n.DB.Model(&node).Updates(data).Error; err != nil {
		return fmt.Errorf("Updates %v", err)
	}
	return nil
}

// UpdateIPAddress to update tge node IP Address
func (n *NodeManager) UpdateIPAddress(ipaddress string, node OsqueryNode) error {
	data := OsqueryNode{
		IPAddress: "",
	}
	if (ipaddress != "") && (ipaddress != node.IPAddress) {
		data.IPAddress = ipaddress
		e := NodeHistoryIPAddress{
			UUID:      node.UUID,
			IPAddress: ipaddress,
			Count:     1,
		}
		if err := n.NewHistoryIPAddress(e); err != nil {
			return fmt.Errorf("newNodeHistoryIPAddress %v", err)
		}
		if err := n.DB.Model(&node).Updates(data).Error; err != nil {
			return fmt.Errorf("Updates %v", err)
		}
	} else {
		if err := n.IncHistoryIPAddress(node.UUID, ipaddress); err != nil {
			return fmt.Errorf("incNodeHistoryIPAddress %v", err)
		}
		if err := n.DB.Model(&node).Update("updated_at", time.Now()).Error; err != nil {
			return fmt.Errorf("Update %v", err)
		}
	}
	return nil
}

// UpdateIPAddressByUUID to update node IP Address by UUID
func (n *NodeManager) UpdateIPAddressByUUID(ipaddress, uuid string) error {
	node, err := n.GetByUUID(uuid)
	if err != nil {
		return fmt.Errorf("getNodeByUUID %v", err)
	}
	return n.UpdateIPAddress(ipaddress, node)
}

// UpdateIPAddressByKey to update node IP Address by node_key
func (n *NodeManager) UpdateIPAddressByKey(ipaddress, nodekey string) error {
	node, err := n.GetByKey(nodekey)
	if err != nil {
		return fmt.Errorf("getNodeByKey %v", err)
	}
	return n.UpdateIPAddress(ipaddress, node)
}

// Create to insert new osquery node generating new node_key
func (n *NodeManager) Create(node *OsqueryNode) error {
	if n.DB.NewRecord(node) {
		if err := n.DB.Create(&node).Error; err != nil {
			return fmt.Errorf("Create %v", err)
		}
		h := NodeHistoryHostname{
			UUID:     node.UUID,
			Hostname: node.Hostname,
		}
		if err := n.NewHistoryHostname(h); err != nil {
			return fmt.Errorf("newNodeHistoryHostname %v", err)
		}
		l := NodeHistoryLocalname{
			UUID:      node.UUID,
			Localname: node.Localname,
		}
		if err := n.NewHistoryLocalname(l); err != nil {
			return fmt.Errorf("newNodeHistoryLocalname %v", err)
		}
		i := NodeHistoryIPAddress{
			UUID:      node.UUID,
			IPAddress: node.IPAddress,
			Count:     1,
		}
		if err := n.NewHistoryIPAddress(i); err != nil {
			return fmt.Errorf("newNodeHistoryIPAddress %v", err)
		}
		// FIXME needs rewriting
		//if err := geoLocationCheckByIPAddress(node.IPAddress); err != nil {
		//	return fmt.Errorf("geoLocationCheckByIPAddress %v", err)
		//}
		u := NodeHistoryUsername{
			UUID:     node.UUID,
			Username: node.Username,
		}
		if err := n.NewHistoryUsername(u); err != nil {
			return fmt.Errorf("newNodeHistoryUsername %v", err)
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	return nil
}

// NewHistoryEntry to insert new entry for the history of Hostnames
func (n *NodeManager) NewHistoryEntry(entry interface{}) error {
	if n.DB.NewRecord(entry) {
		if err := n.DB.Create(&entry).Error; err != nil {
			return fmt.Errorf("Create newNodeHistoryEntry %v", err)
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	return nil
}

// NewHistoryHostname to insert new entry for the history of Hostnames
func (n *NodeManager) NewHistoryHostname(entry NodeHistoryHostname) error {
	if n.DB.NewRecord(entry) {
		if err := n.DB.Create(&entry).Error; err != nil {
			return fmt.Errorf("Create newNodeHistoryHostname %v", err)
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	return nil
}

// NewHistoryLocalname to insert new entry for the history of Localnames
func (n *NodeManager) NewHistoryLocalname(entry NodeHistoryLocalname) error {
	if n.DB.NewRecord(entry) {
		if err := n.DB.Create(&entry).Error; err != nil {
			return fmt.Errorf("Create newNodeHistoryLocalname %v", err)
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	return nil
}

// NewHistoryUsername to insert new entry for the history of Usernames
func (n *NodeManager) NewHistoryUsername(entry NodeHistoryUsername) error {
	if n.DB.NewRecord(entry) {
		if err := n.DB.Create(&entry).Error; err != nil {
			return fmt.Errorf("Create newNodeHistoryUsername %v", err)
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	return nil
}

// NewHistoryIPAddress to insert new entry for the history of IP Addresses
func (n *NodeManager) NewHistoryIPAddress(entry NodeHistoryIPAddress) error {
	if n.DB.NewRecord(entry) {
		if err := n.DB.Create(&entry).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	return nil
}

// GetHistoryIPAddress to retrieve the History IP Address record by UUID and the IP Address
func (n *NodeManager) GetHistoryIPAddress(uuid, ipaddress string) (NodeHistoryIPAddress, error) {
	var nodeip NodeHistoryIPAddress
	if err := n.DB.Where("uuid = ? AND ip_address = ?", uuid, ipaddress).Order("updated_at").First(&nodeip).Error; err != nil {
		return nodeip, err
	}
	return nodeip, nil
}

// IncHistoryIPAddress to increase the count for this IP Address
func (n *NodeManager) IncHistoryIPAddress(uuid, ipaddress string) error {
	nodeip, err := n.GetHistoryIPAddress(uuid, ipaddress)
	if err != nil {
		return fmt.Errorf("getNodeHistoryIPAddress %v", err)
	}
	if err := n.DB.Model(&nodeip).Update("count", nodeip.Count+1).Error; err != nil {
		return fmt.Errorf("Update %v", err)
	}
	return nil
}

// Archive to archive osquery node by UUID
func (n *NodeManager) Archive(uuid, trigger string) error {
	node, err := n.GetByUUID(uuid)
	if err != nil {
		return fmt.Errorf("getNodeByUUID %v", err)
	}
	archivedNode := nodeArchiveFromNode(node, trigger)
	if n.DB.NewRecord(archivedNode) {
		if err := n.DB.Create(&archivedNode).Error; err != nil {
			return fmt.Errorf("Create %v", err)
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	return nil
}

// UpdateByUUID to update an existing node record by UUID
func (n *NodeManager) UpdateByUUID(data OsqueryNode, uuid string) error {
	node, err := n.GetByUUID(uuid)
	if err != nil {
		return fmt.Errorf("getNodeByUUID %v", err)
	}
	if err := n.DB.Model(&node).Updates(data).Error; err != nil {
		return fmt.Errorf("Updates %v", err)
	}
	return nil
}

// ArchiveDeleteByUUID to archive and delete an existing node record by UUID
func (n *NodeManager) ArchiveDeleteByUUID(uuid string) error {
	node, err := n.GetByUUID(uuid)
	if err != nil {
		return fmt.Errorf("getNodeByUUID %v", err)
	}
	archivedNode := nodeArchiveFromNode(node, "delete")
	if n.DB.NewRecord(archivedNode) {
		if err := n.DB.Create(&archivedNode).Error; err != nil {
			return fmt.Errorf("Create %v", err)
		}
	} else {
		return fmt.Errorf("n.DB.NewRecord did not return true")
	}
	if err := n.DB.Unscoped().Delete(&node).Error; err != nil {
		return fmt.Errorf("Delete %v", err)
	}
	return nil
}

// RefreshLastEventByUUID to refresh the last status log for this node
func (n *NodeManager) RefreshLastEventByUUID(uuid, event string) error {
	node, err := n.GetByUUID(uuid)
	if err != nil {
		return fmt.Errorf("getNodeByUUID %v", err)
	}
	if err := n.DB.Model(&node).Update(event, time.Now()).Error; err != nil {
		return fmt.Errorf("Update %v", err)
	}
	return nil
}

// RefreshLastEventByKey to refresh the last status log for this node
func (n *NodeManager) RefreshLastEventByKey(nodeKey, event string) error {
	node, err := n.GetByKey(nodeKey)
	if err != nil {
		return err
	}
	if err := n.DB.Model(&node).Update(event, time.Now()).Error; err != nil {
		return fmt.Errorf("Update %v", err)
	}
	return nil
}

// RefreshLastStatus to refresh the last status log for this node
func (n *NodeManager) RefreshLastStatus(uuid string) error {
	return n.RefreshLastEventByUUID(uuid, "last_status")
}

// RefreshLastResult to refresh the last result log for this node
func (n *NodeManager) RefreshLastResult(uuid string) error {
	return n.RefreshLastEventByUUID(uuid, "last_result")
}

// RefreshLastConfig to refresh the last configuration for this node
func (n *NodeManager) RefreshLastConfig(nodeKey string) error {
	return n.RefreshLastEventByKey(nodeKey, "last_config")
}

// RefreshLastQueryRead to refresh the last on-demand query read for this node
func (n *NodeManager) RefreshLastQueryRead(nodeKey string) error {
	return n.RefreshLastEventByKey(nodeKey, "last_query_read")
}

// RefreshLastQueryWrite to refresh the last on-demand query write for this node
func (n *NodeManager) RefreshLastQueryWrite(uuid string) error {
	return n.RefreshLastEventByUUID(uuid, "last_query_write")
}

// Helper to convert an enrolled osquery node into an archived osquery node
func nodeArchiveFromNode(node OsqueryNode, trigger string) ArchiveOsqueryNode {
	return ArchiveOsqueryNode{
		NodeKey:         node.NodeKey,
		UUID:            node.UUID,
		Trigger:         trigger,
		Platform:        node.Platform,
		PlatformVersion: node.PlatformVersion,
		OsqueryVersion:  node.OsqueryVersion,
		Hostname:        node.Hostname,
		Localname:       node.Localname,
		IPAddress:       node.IPAddress,
		Username:        node.Username,
		OsqueryUser:     node.OsqueryUser,
		Environment:     node.Environment,
		CPU:             node.CPU,
		Memory:          node.Memory,
		HardwareSerial:  node.HardwareSerial,
		DaemonHash:      node.DaemonHash,
		ConfigHash:      node.ConfigHash,
		RawEnrollment:   node.RawEnrollment,
		LastStatus:      node.LastStatus,
		LastResult:      node.LastResult,
		LastConfig:      node.LastConfig,
		LastQueryRead:   node.LastQueryRead,
		LastQueryWrite:  node.LastQueryWrite,
	}
}
