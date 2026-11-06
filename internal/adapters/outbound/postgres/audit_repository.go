package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct{ db *pgxpool.Pool }

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) Append(ctx context.Context, item domain.AuditLog) error {
	_, err := r.db.Exec(ctx, `INSERT INTO audit_logs(actor_id, action, entity, entity_id, result, origin)
		VALUES(NULLIF($1,'')::uuid, $2, $3, NULLIF($4,'')::uuid, $5, NULLIF($6,''))`,
		item.ActorID, item.Action, item.Entity, item.EntityID, item.Result, item.Origin)
	return err
}

func (r *AuditRepository) List(ctx context.Context, filter ports.AuditFilter) ([]domain.AuditLog, error) {
	where := []string{"1=1"}
	args := make([]any, 0, 5)
	add := func(condition string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if strings.TrimSpace(filter.ActorID) != "" {
		add("actor_id::text = $%d", strings.TrimSpace(filter.ActorID))
	}
	if strings.TrimSpace(filter.Action) != "" {
		add("action = $%d", strings.TrimSpace(filter.Action))
	}
	if strings.TrimSpace(filter.Entity) != "" {
		add("entity = $%d", strings.TrimSpace(filter.Entity))
	}
	if filter.From != nil {
		add("created_at >= $%d", *filter.From)
	}
	if filter.To != nil {
		add("created_at <= $%d", *filter.To)
	}
	rows, err := r.db.Query(ctx, `SELECT id, COALESCE(actor_id::text,''), action, entity,
		COALESCE(entity_id::text,''), created_at, result, COALESCE(origin,'')
		FROM audit_logs WHERE `+strings.Join(where, " AND ")+` ORDER BY created_at DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logs := make([]domain.AuditLog, 0)
	for rows.Next() {
		var item domain.AuditLog
		if err := rows.Scan(&item.ID, &item.ActorID, &item.Action, &item.Entity, &item.EntityID, &item.Date, &item.Result, &item.Origin); err != nil {
			return nil, err
		}
		logs = append(logs, item)
	}
	return logs, rows.Err()
}
