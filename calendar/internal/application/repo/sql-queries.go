package repo

const createEvent = `INSERT INTO events (
                    id, title, start_date_event, creation_date, end_date_event, 
                    description_event, user_id, time_for_notification, rq_tm) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO NOTHING
RETURNING id;`

const getEventsByPeriod = `SELECT * FROM events 
WHERE start_date_event >= $1 and end_date_event <= $2`

const deleteEvent = `DELETE FROM events WHERE id = $1`

const deleteOldEvents = `DELETE FROM events
		WHERE creation_date < now() - make_interval(days => $1)`

// OUTBOX
const insertOutboxQuery = `
INSERT INTO outbox_event (
  aggregate_id, aggregate_type, event_type, payload, status, attempts, next_attempt_at, created_at
) VALUES ($1,$2,$3, ($4)::jsonb, $5, 0, now(), now())
RETURNING id
`

const reserveBatchSQL = `
WITH picked AS (
	SELECT id
  	FROM outbox_event
  	WHERE status IN ('NEW','FAILED')
		AND next_attempt_at <= now()
    	AND attempts < $3             
  	ORDER BY id
  	FOR UPDATE SKIP LOCKED
	LIMIT $2
)
UPDATE outbox_event AS o
SET next_attempt_at = now() + $1::interval
FROM picked
WHERE o.id = picked.id
RETURNING o.id, o.aggregate_id, o.aggregate_type, o.event_type, o.payload, o.status, o.attempts, o.next_attempt_at, o.created_at;
`

const markFailedSQL = `
UPDATE outbox_event
SET status=$2, attempts=attempts+1, next_attempt_at=$3
WHERE id=$1`

const markGaveUpSQL = `
UPDATE outbox_event
SET status=$2, attempts=attempts+1, next_attempt_at = now()
WHERE id=$1
`

const markSentSQL = `UPDATE outbox_event SET status=$2 WHERE id=$1`
