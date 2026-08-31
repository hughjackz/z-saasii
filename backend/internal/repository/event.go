package repository

import "github.com/yourorg/csms-backend/internal/model"

// SaveEvent stores an event in the database for historical queries.
func SaveEvent(ev *model.Event, tenantID string) {
	_, _ = DB.Exec(
		`INSERT INTO event_log (tenant_id, time, level, device, message) VALUES (?, ?, ?, ?, ?)`,
		tenantID, ev.Time, ev.Level, ev.Device, ev.Message)
}

// QueryEvents returns events matching the given filters.
func QueryEvents(tenantID, date, level string, limit int) ([]*model.Event, error) {
	if limit <= 0 {
		limit = 200
	}
	q := "SELECT time, level, device, message FROM event_log WHERE 1=1"
	args := []interface{}{}
	if tenantID != "" {
		q += " AND tenant_id=?"
		args = append(args, tenantID)
	}
	if date != "" {
		q += " AND DATE(time)=?"
		args = append(args, date)
	}
	if level != "" {
		q += " AND level=?"
		args = append(args, level)
	}
	q += " ORDER BY time DESC LIMIT ?"
	args = append(args, limit)

	var events []*model.Event
	if err := DB.Select(&events, q, args...); err != nil {
		return nil, err
	}
	if events == nil {
		events = []*model.Event{}
	}
	return events, nil
}
