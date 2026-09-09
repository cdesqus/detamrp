package bom

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/shopspring/decimal"
)

// Node describes usage per one base unit of its parent, never per Kanban.
type Node struct {
	ItemID       string          `json:"itemId"`
	Kind         string          `json:"kind"`
	ItemCode     string          `json:"itemCode"`
	Name         string          `json:"name"`
	Unit         string          `json:"unit"`
	Discrete     bool            `json:"discrete"`
	Usage        decimal.Decimal `json:"usage"`
	BOMID        string          `json:"bomId,omitempty"`
	Revision     int             `json:"revision,omitempty"`
	UnitPrice    decimal.Decimal `json:"unitPrice"`
	Currency     string          `json:"currency,omitempty"`
	QtyPerKanban decimal.Decimal `json:"qtyPerKanban"`
	Children     []Node          `json:"children,omitempty"`
}
type Snapshot struct {
	Version int  `json:"version"`
	Root    Node `json:"root"`
}
type RequirementNode struct {
	Path       string          `json:"path"`
	ParentPath string          `json:"parentPath"`
	ItemID     string          `json:"itemId"`
	Kind       string          `json:"kind"`
	ItemCode   string          `json:"itemCode"`
	Name       string          `json:"name"`
	Unit       string          `json:"unit"`
	Usage      decimal.Decimal `json:"usage"`
	Quantity   decimal.Decimal `json:"quantity"`
	Terminal   bool            `json:"terminal"`
	Depth      int             `json:"depth"`
}
type MaterialRequirement struct {
	ItemID           string          `json:"itemId"`
	ItemCode         string          `json:"itemCode"`
	Name             string          `json:"name"`
	Unit             string          `json:"unit"`
	Quantity         decimal.Decimal `json:"quantity"`
	UnitPrice        decimal.Decimal `json:"unitPrice"`
	Currency         string          `json:"currency"`
	Value            decimal.Decimal `json:"value"`
	QtyPerKanban     decimal.Decimal `json:"qtyPerKanban"`
	KanbanEquivalent decimal.Decimal `json:"kanbanEquivalent"`
	PurchaseKanban   decimal.Decimal `json:"purchaseKanban"`
	Paths            []string        `json:"paths"`
}
type RequirementResult struct {
	Nodes        []RequirementNode          `json:"nodes"`
	Materials    []MaterialRequirement      `json:"materials"`
	Totals       map[string]decimal.Decimal `json:"totals"`
	CostComplete bool                       `json:"costComplete"`
}

var maxStoredDecimal = decimal.RequireFromString("99999999999999.999999")

func validQuantity(q decimal.Decimal) bool {
	return !q.IsNegative() && q.LessThanOrEqual(maxStoredDecimal) && q.Equal(q.Round(6))
}

// BuildSnapshot validates and deep-copies the graph so later master edits cannot change it.
func BuildSnapshot(root Node) (Snapshot, error) {
	if root.Kind != "FG" && root.Kind != "RAW_MATERIAL" {
		return Snapshot{}, fmt.Errorf("invalid output kind")
	}
	if err := validateNode(root, map[string]bool{}, 0); err != nil {
		return Snapshot{}, err
	}
	data, err := json.Marshal(root)
	if err != nil {
		return Snapshot{}, err
	}
	var saved Node
	if err = json.Unmarshal(data, &saved); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Version: 1, Root: saved}, nil
}
func validateNode(n Node, stack map[string]bool, depth int) error {
	if depth > 64 {
		return fmt.Errorf("BOM exceeds 64 levels")
	}
	if n.ItemID == "" || n.ItemCode == "" || n.Unit == "" {
		return fmt.Errorf("item identity, code and unit are required")
	}
	if n.Kind != "FG" && n.Kind != "RAW_MATERIAL" {
		return fmt.Errorf("invalid item kind for %s", n.ItemCode)
	}
	key := n.Kind + ":" + n.ItemID
	if stack[key] {
		return fmt.Errorf("circular BOM at %s", n.ItemCode)
	}
	if !n.Usage.IsPositive() || !validQuantity(n.Usage) {
		return fmt.Errorf("invalid usage for %s", n.ItemCode)
	}
	if n.Discrete && !n.Usage.Equal(n.Usage.Truncate(0)) {
		return fmt.Errorf("whole units required for %s", n.ItemCode)
	}
	if n.UnitPrice.IsNegative() || n.UnitPrice.GreaterThan(maxStoredDecimal) {
		return fmt.Errorf("invalid price for %s", n.ItemCode)
	}
	if n.QtyPerKanban.IsNegative() {
		return fmt.Errorf("invalid Kanban factor for %s", n.ItemCode)
	}
	if len(n.Children) == 0 && n.Currency == "" {
		return fmt.Errorf("material currency missing for %s", n.ItemCode)
	}
	stack[key] = true
	defer delete(stack, key)
	for _, child := range n.Children {
		if err := validateNode(child, stack, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// Calculate uses only captured data. Zero is useful for fully delivered SO lines.
func Calculate(snapshot Snapshot, quantity decimal.Decimal) (RequirementResult, error) {
	result := RequirementResult{Nodes: []RequirementNode{}, Materials: []MaterialRequirement{}, Totals: map[string]decimal.Decimal{}, CostComplete: true}
	if snapshot.Version != 1 {
		return result, fmt.Errorf("unsupported calculation snapshot version")
	}
	if !validQuantity(quantity) {
		return result, fmt.Errorf("invalid requested quantity")
	}
	if err := validateNode(snapshot.Root, map[string]bool{}, 0); err != nil {
		return result, err
	}
	if quantity.IsZero() {
		return result, nil
	}
	grouped := map[string]*MaterialRequirement{}
	var walk func(Node, decimal.Decimal, string, string, int) error
	walk = func(n Node, parent decimal.Decimal, path, parentPath string, depth int) error {
		q := parent.Mul(n.Usage)
		if !validQuantity(q) {
			return fmt.Errorf("quantity exceeds precision or range for %s", n.ItemCode)
		}
		if n.Discrete && !q.Equal(q.Truncate(0)) {
			return fmt.Errorf("whole units required for %s", n.ItemCode)
		}
		terminal := len(n.Children) == 0
		result.Nodes = append(result.Nodes, RequirementNode{Path: path, ParentPath: parentPath, ItemID: n.ItemID, Kind: n.Kind, ItemCode: n.ItemCode, Name: n.Name, Unit: n.Unit, Usage: n.Usage, Quantity: q, Terminal: terminal, Depth: depth})
		if terminal {
			key := n.ItemID + "|" + n.Unit + "|" + n.Currency + "|" + n.UnitPrice.String() + "|" + n.QtyPerKanban.String()
			row := grouped[key]
			if row == nil {
				row = &MaterialRequirement{ItemID: n.ItemID, ItemCode: n.ItemCode, Name: n.Name, Unit: n.Unit, UnitPrice: n.UnitPrice, Currency: n.Currency, QtyPerKanban: n.QtyPerKanban}
				grouped[key] = row
			}
			row.Quantity = row.Quantity.Add(q)
			row.Paths = append(row.Paths, path)
			if !validQuantity(row.Quantity) {
				return fmt.Errorf("aggregate quantity overflow for %s", n.ItemCode)
			}
			if n.UnitPrice.IsZero() {
				result.CostComplete = false
			}
		}
		for i, child := range n.Children {
			if err := walk(child, q, path+"."+strconv.Itoa(i), path, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(snapshot.Root, quantity, "0", "", 0); err != nil {
		return result, err
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		row := grouped[key]
		row.Value = row.Quantity.Mul(row.UnitPrice).Round(6)
		if !validQuantity(row.Value) {
			return result, fmt.Errorf("material value overflow for %s", row.ItemCode)
		}
		if row.QtyPerKanban.IsPositive() {
			row.KanbanEquivalent = row.Quantity.DivRound(row.QtyPerKanban, 6)
			row.PurchaseKanban = row.Quantity.DivRound(row.QtyPerKanban, 16).Ceil()
		}
		result.Materials = append(result.Materials, *row)
		result.Totals[row.Currency] = result.Totals[row.Currency].Add(row.Value)
		if !validQuantity(result.Totals[row.Currency]) {
			return result, fmt.Errorf("currency total overflow")
		}
	}
	return result, nil
}
