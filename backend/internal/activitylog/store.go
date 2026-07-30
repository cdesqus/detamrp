package activitylog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"order-stock/backend/internal/database"
)

var filterModules = []string{"DATA_MASTER", "PROCUREMENT", "LOGISTICS", "RECEIVING", "OUTGOING", "INVENTORY", "SETTINGS"}
var filterActions = []string{
	"CREATED", "UPDATED", "ACTIVATED", "DEACTIVATED", "SUBMITTED", "APPROVED",
	"REJECTED", "CANCELLED", "ISSUED", "COMPLETED", "RECEIVED", "MOVED",
	"COMPANY_LOGO_UPDATED", "LOGIN_BACKGROUND_UPDATED",
}

type Store struct{ db *database.Pool }

func NewStore(db *database.Pool) *Store { return &Store{db: db} }

func (s *Store) List(ctx context.Context, actor Actor, query Query) (page Page, err error) {
	query.TenantID = actor.TenantID
	page = Page{
		Items:    []Item{},
		Page:     query.Page,
		PageSize: query.PageSize,
		Filters: FilterOptions{
			Actors:  []ActorOption{},
			Modules: append([]string(nil), filterModules...),
			Actions: append([]string(nil), filterActions...),
		},
	}
	err = database.WithTenant(ctx, s.db, database.TenantContext{TenantID: actor.TenantID, UserID: actor.UserID}, func(tx database.TenantTx) error {
		sql, args := buildListQuery(query)
		rows, queryErr := tx.Query(ctx, sql, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()
		for rows.Next() {
			var item Item
			var beforeJSON, afterJSON []byte
			if err := rows.Scan(
				&item.ID,
				&item.OccurredAt,
				&item.ActorUserID,
				&item.ActorName,
				&item.Module,
				&item.Action,
				&item.TargetType,
				&item.TargetID,
				&item.TargetCode,
				&beforeJSON,
				&afterJSON,
				&page.Total,
			); err != nil {
				return err
			}
			if err := decodeSnapshot(beforeJSON, &item.Before); err != nil {
				return err
			}
			if err := decodeSnapshot(afterJSON, &item.After); err != nil {
				return err
			}
			page.Items = append(page.Items, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return loadActorOptions(ctx, tx, actor.TenantID, &page.Filters.Actors)
	})
	return page, err
}

func buildListQuery(query Query) (string, []any) {
	args := []any{query.TenantID, query.From, query.To.AddDate(0, 0, 1)}
	where := []string{"tenant_id=$1", "occurred_at >= $2", "occurred_at < $3"}
	add := func(column string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if query.UserID != uuid.Nil {
		add("actor_user_id", query.UserID)
	}
	if query.Module != "" {
		add("module", query.Module)
	}
	if query.Action != "" {
		add("action", query.Action)
	}
	args = append(args, query.PageSize, query.Offset())
	return fmt.Sprintf(`
SELECT id,occurred_at,actor_user_id,actor_name,module,action,target_type,target_id,target_code,
       before_data,after_data,count(*) OVER()
FROM activity_logs
WHERE %s
ORDER BY occurred_at DESC,id DESC
LIMIT $%d OFFSET $%d`, strings.Join(where, " AND "), len(args)-1, len(args)), args
}

func loadActorOptions(ctx context.Context, tx database.TenantTx, tenantID uuid.UUID, target *[]ActorOption) error {
	rows, err := tx.Query(ctx, `
SELECT actor_user_id,max(actor_name)
FROM activity_logs
WHERE tenant_id=$1 AND actor_user_id IS NOT NULL
GROUP BY actor_user_id
ORDER BY max(actor_name),actor_user_id`, tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var option ActorOption
		if err := rows.Scan(&option.ID, &option.Name); err != nil {
			return err
		}
		*target = append(*target, option)
	}
	return rows.Err()
}

func decodeSnapshot(raw []byte, target *map[string]any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, target)
}
